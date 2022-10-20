/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"data-space.liqo.io/consts"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=namespaces/finalizers,verbs=update

//+kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	nsName := req.NamespacedName
	klog.Infof("Reconcile Namespace %q", nsName.Name)

	// Network policy namespaced name
	npNsName := types.NamespacedName{
		Namespace: nsName.Name,
		Name:      consts.NetworkPolicyName,
	}
	// Config map namespaced name
	cmNsName := types.NamespacedName{
		Namespace: nsName.Name,
		Name:      consts.ConfigMapName,
	}

	namespace := corev1.Namespace{}
	if err := r.Get(ctx, nsName, &namespace); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Namespace %q not found: trying to delete NetworkPolicy %q and ConfigMap %q", nsName.Name, npNsName, cmNsName)
			// Delete relevant NetworkPolicy and ConfigMap if found
			if err := r.deleteNetworkPolicy(ctx, npNsName); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.deleteConfigMap(ctx, cmNsName); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, err
	}

	// Intercept if the object is under deletion and return no errors
	if !namespace.ObjectMeta.DeletionTimestamp.IsZero() {
		klog.Infof("Namespace %q is under deletion. Relevant resources are going to be deleted as well.", nsName.Name)
		return ctrl.Result{}, nil
	}

	// Check the "apply network policy" label
	if v, ok := namespace.Labels[consts.DataSpaceApplyNetpolLabel]; ok && v == "false" {
		// Namespace label set to false: delete relevant NetworkPolicy if found
		klog.Infof("Namespace %q contains label %q: trying to delete NetworkPolicy %q", nsName.Name, fmt.Sprintf("%s:%s", consts.DataSpaceApplyNetpolLabel, "false"), npNsName)
		if err := r.deleteNetworkPolicy(ctx, npNsName); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// Namespace label not set to false: create NetworkPolicy if not found
		networkPolicy := forgeNetworkPolicy(nsName.Name)
		if err := r.createNetworkPolicy(ctx, nsName.Name, networkPolicy); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check the "apply webhook" label
	if v, ok := namespace.Labels[consts.DataSpaceApplyWebhookLabel]; ok && v == "false" {
		// Namespace label set to false: delete relevant ConfigMap if found
		klog.Infof("Namespace %q contains label %q: trying to delete ConfigMap %q", nsName.Name, fmt.Sprintf("%s:%s", consts.DataSpaceApplyWebhookLabel, "false"), cmNsName)
		if err := r.deleteConfigMap(ctx, cmNsName); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		// Namespace label not set to false: create ConfigMap if not found
		configMap, err := forgeConfigMap(nsName.Name)
		if err != nil {
			return ctrl.Result{}, err
		}
		if err := r.createConfigMap(ctx, nsName.Name, configMap); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// deleteNetworkPolicy deletes the NetworkPolicy resource in the reconciled namespace
func (r *NamespaceReconciler) deleteNetworkPolicy(ctx context.Context, nsName types.NamespacedName) error {
	var networkPolicy netv1.NetworkPolicy
	if err := r.Client.Get(ctx, nsName, &networkPolicy); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, nsName.Namespace)
		} else {
			klog.Errorf("Error while getting NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, nsName.Namespace)
		}
		return err
	}

	if err := r.Client.Delete(ctx, &networkPolicy); err != nil {
		klog.Errorf("Error while deleting NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, nsName.Namespace)
	}
	klog.Infof("Deleted NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, nsName.Namespace)
	return nil
}

// deleteConfigMap deletes the ConfigMap resource in the reconciled namespace
func (r *NamespaceReconciler) deleteConfigMap(ctx context.Context, nsName types.NamespacedName) error {
	var configMap corev1.ConfigMap
	if err := r.Client.Get(ctx, nsName, &configMap); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found ConfigMap %q in namespace %q", consts.ConfigMapName, nsName.Namespace)
		} else {
			klog.Errorf("Error while getting ConfigMap %q in namespace %q", consts.ConfigMapName, nsName.Namespace)
		}
		return err
	}

	if err := r.Client.Delete(ctx, &configMap); err != nil {
		klog.Errorf("Error while deleting ConfigMap %q in namespace %q", consts.ConfigMapName, nsName.Namespace)
	}
	klog.Infof("Deleted ConfigMap %q in namespace %q", consts.ConfigMapName, nsName.Namespace)
	return nil
}

// createNetworkPolicy creates a NetworkPolicy resource in the reconciled namespace
func (r *NamespaceReconciler) createNetworkPolicy(ctx context.Context, namespaceName string, networkPolicy *netv1.NetworkPolicy) error {
	if err := r.Client.Create(ctx, networkPolicy); err != nil {
		err = client.IgnoreAlreadyExists(err)
		if err == nil {
			klog.Infof("NetworkPolicy %q already exists in namespace %q", consts.NetworkPolicyName, namespaceName)
		} else {
			klog.Errorf("Error while creating NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, namespaceName)
		}
		return err
	}
	klog.Infof("Created NetworkPolicy %q in namespace %q", consts.NetworkPolicyName, namespaceName)
	return nil
}

// createConfigMap creates a ConfigMap resource in the reconciled namespace
func (r *NamespaceReconciler) createConfigMap(ctx context.Context, namespaceName string, configMap *corev1.ConfigMap) error {
	if err := r.Client.Create(ctx, configMap); err != nil {
		err = client.IgnoreAlreadyExists(err)
		if err == nil {
			klog.Infof("ConfigMap %q already exists in namespace %q", consts.ConfigMapName, namespaceName)
		} else {
			klog.Errorf("Error while creating ConfigMap %q in namespace %q", consts.ConfigMapName, namespaceName)
		}
		return err
	}
	klog.Infof("Created ConfigMap %q in namespace %q", consts.ConfigMapName, namespaceName)
	return nil
}

// forgeNetworkPolicy builds a NetworkPolicy object
func forgeNetworkPolicy(namespaceName string) *netv1.NetworkPolicy {
	return &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.NetworkPolicyName,
			Namespace: namespaceName,
		},
		Spec: netv1.NetworkPolicySpec{
			// For matching pods in the NetworkPolicy's namespace
			PodSelector: metav1.LabelSelector{
				// Empty: match all
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
				netv1.PolicyTypeEgress,
			},
			Ingress: []netv1.NetworkPolicyIngressRule{{
				From: []netv1.NetworkPolicyPeer{
					// Allow all sources
				},
			}},
			Egress: []netv1.NetworkPolicyEgressRule{{
				To: []netv1.NetworkPolicyPeer{
					// OR-ed selectors
					{
						// For pods in matching namespaces
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								consts.DataSpaceNetpolAllowLabel: "true",
							},
						},
					},
					{
						// For matching pods in the NetworkPolicy's namespace
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								consts.DataSpaceNetpolAllowLabel: "true",
							},
						},
					},
					{
						// AND-ed selectors
						// For matching pods in matching namespaces (to allow for DNS queries)
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								corev1.LabelMetadataName: consts.KUBE_SYSTEM,
							},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								consts.K8sAppLabel: consts.KUBE_DNS,
							},
						},
					},
				},
			}},
		},
	}
}

// forgeConfigMap builds a ConfigMap object
func forgeConfigMap(namespaceName string) (*corev1.ConfigMap, error) {
	envoyConfig := forgeEnvoyConfig()

	// Encode object into yaml
	marshaledEnvoyConfig, err := yaml.Marshal(envoyConfig)
	if err != nil {
		klog.Fatal("Could not marshal yaml, error: %v", err)
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.ConfigMapName,
			Namespace: namespaceName,
		},
		Data: map[string]string{
			"keys": string(marshaledEnvoyConfig),
		},
	}, nil
}

// forgeEnvoyConfig builds an EnvoyConfig object
func forgeEnvoyConfig() *EnvoyConfig {
	return &EnvoyConfig{
		Admin: Admin{
			Address: Address{
				SocketAddress: SocketAddress{
					Address:   consts.LOCALHOST_ADDR,
					PortValue: consts.AdminPort,
				},
			},
		},

		StaticResources: StaticResources{
			Listeners: []Listener{
				// Egress TCP listener
				{
					Name: "egress_tcp_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   consts.LOCALHOST_ADDR,
							PortValue: consts.EgressTcpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: consts.TcpProxyTypeName,
									TypedConfig: TypedConfig{
										Type:       consts.TcpProxyTypeUrl,
										StatPrefix: consts.EgressTcpStatPrefixName,
										Cluster:    consts.EgressClusterName,
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: consts.AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: consts.AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
					ListenerFilters: []NameAndConfig{
						{
							Name: consts.OriginalDstListenerFilterTypeName,
							TypedConfig: TypedConfig{
								Type: consts.OriginalDstListenerFilterTypeUrl,
							},
						},
					},
				},
				// Ingress TCP listener
				{
					Name: "ingress_tcp_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   consts.ANY_ADDR,
							PortValue: consts.IngressTcpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: consts.TcpProxyTypeName,
									TypedConfig: TypedConfig{
										Type:       consts.TcpProxyTypeUrl,
										StatPrefix: consts.IngressTcpStatPrefixName,
										Cluster:    consts.IngressClusterName,
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: consts.AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: consts.AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
					ListenerFilters: []NameAndConfig{
						{
							Name: consts.OriginalDstListenerFilterTypeName,
							TypedConfig: TypedConfig{
								Type: consts.OriginalDstListenerFilterTypeUrl,
							},
						},
					},
				},
				// Egress HTTP listener
				{
					Name: "egress_http_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   consts.LOCALHOST_ADDR,
							PortValue: consts.EgressHttpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: consts.HttpConnectionManagerTypeName,
									TypedConfig: TypedConfig{
										Type:       consts.HttpConnectionManagerTypeUrl,
										StatPrefix: consts.EgressHttpStatPrefixName,
										HttpFilters: []NameAndConfig{
											// Forward
											{
												Name: consts.DynamicForwardProxyFilterTypeName,
												TypedConfig: TypedConfig{
													Type: consts.DynamicForwardProxyFilterTypeUrl,
													DnsCacheConfig: DnsCacheConfig{
														Name:            consts.DnsCacheConfigName,
														DnsLookupFamily: consts.Ipv4Only,
													},
												},
											},
											// Router
											{
												Name: consts.RouterTypeName,
												TypedConfig: TypedConfig{
													Type: consts.RouterTypeUrl,
												},
											},
										},
										RouteConfig: RouteConfig{
											Name: "egress_route_config",
											VirtualHosts: []VirtualHost{
												{
													Name:    "egress_forward_host",
													Domains: []string{"*"},
													Routes: []RouteEntry{
														{
															Name: "egress_forward_route",
															Match: Match{
																Prefix: "/",
															},
															Route: Route{
																Cluster: consts.EgressForwardClusterName,
															},
														},
													},
												},
											},
										},
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: consts.AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: consts.AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
				},
				// Ingress HTTP listener
				{
					Name: "ingress_http_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   consts.ANY_ADDR,
							PortValue: consts.IngressHttpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: consts.HttpConnectionManagerTypeName,
									TypedConfig: TypedConfig{
										Type:       consts.HttpConnectionManagerTypeUrl,
										StatPrefix: consts.IngressHttpStatPrefixName,
										HttpFilters: []NameAndConfig{
											// Forward
											{
												Name: consts.DynamicForwardProxyFilterTypeName,
												TypedConfig: TypedConfig{
													Type: consts.DynamicForwardProxyFilterTypeUrl,
													DnsCacheConfig: DnsCacheConfig{
														Name:            consts.DnsCacheConfigName,
														DnsLookupFamily: consts.Ipv4Only,
													},
												},
											},
											// Router
											{
												Name: consts.RouterTypeName,
												TypedConfig: TypedConfig{
													Type: consts.RouterTypeUrl,
												},
											},
										},
										RouteConfig: RouteConfig{
											Name: "ingress_route_config",
											VirtualHosts: []VirtualHost{
												{
													Name:    "ingress_forward_host",
													Domains: []string{"*"},
													Routes: []RouteEntry{
														{
															Name: "ingress_forward_route",
															Match: Match{
																Prefix: "/",
															},
															Route: Route{
																Cluster: consts.IngressForwardClusterName,
															},
														},
													},
												},
											},
										},
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: consts.AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: consts.AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
				},
			},
			Clusters: []Cluster{
				// Egress cluster
				{
					Name:            consts.EgressClusterName,
					DnsLookupFamily: consts.Ipv4Only,
					Type:            consts.OriginalDstType,
					LbPolicy:        consts.OriginalDstLbPolicy,
					ConnectTimeout:  "6s",
					OriginalDstLbConfig: OriginalDstLbConfig{
						UseHttpHeader: true,
					},
				},
				// Ingress cluster
				{
					Name:            consts.IngressClusterName,
					DnsLookupFamily: consts.Ipv4Only,
					Type:            consts.OriginalDstType,
					LbPolicy:        consts.OriginalDstLbPolicy,
					ConnectTimeout:  "6s",
					OriginalDstLbConfig: OriginalDstLbConfig{
						UseHttpHeader: true,
					},
				},
				// Egress forward cluster
				{
					Name:           consts.EgressForwardClusterName,
					LbPolicy:       consts.OriginalDstLbPolicy,
					ConnectTimeout: "6s",
					ClusterType: NameAndConfig{
						Name: consts.DynamicForwardProxyClusterTypeName,
						TypedConfig: TypedConfig{
							Type: consts.DynamicForwardProxyClusterTypeUrl,
							DnsCacheConfig: DnsCacheConfig{
								Name:            consts.DnsCacheConfigName,
								DnsLookupFamily: consts.Ipv4Only,
							},
						},
					},
				},
				// Ingress forward cluster
				{
					Name:           consts.IngressForwardClusterName,
					LbPolicy:       consts.OriginalDstLbPolicy,
					ConnectTimeout: "6s",
					ClusterType: NameAndConfig{
						Name: consts.DynamicForwardProxyClusterTypeName,
						TypedConfig: TypedConfig{
							Type: consts.DynamicForwardProxyClusterTypeUrl,
							DnsCacheConfig: DnsCacheConfig{
								Name:            consts.DnsCacheConfigName,
								DnsLookupFamily: consts.Ipv4Only,
							},
						},
					},
				},
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// namespacePredicate selects those namespaces matching the provided label
	namespacePredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{{
			Key:      consts.RemoteClusterIdLabel,
			Operator: metav1.LabelSelectorOpExists,
		}},
	})
	if err != nil {
		klog.Error(err)
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}, builder.WithPredicates(namespacePredicate)).
		Complete(r)
}
