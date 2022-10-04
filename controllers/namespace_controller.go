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

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

	npNsName := types.NamespacedName{
		Namespace: nsName.Name,
		Name:      networkPolicyName,
	}
	cmNsName := types.NamespacedName{
		Namespace: nsName.Name,
		Name:      configMapName,
	}

	namespace := corev1.Namespace{}
	if err := r.Get(ctx, nsName, &namespace); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found Namespace %q", nsName.Name)
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

	// Intercept if the object is under deletion
	if !namespace.ObjectMeta.DeletionTimestamp.IsZero() {
		klog.Infof("Namespace %q is under deletion. Relevant resources are going to be deleted as well.", nsName.Name)
		return ctrl.Result{}, nil
	}

	if v, ok := namespace.Labels[dataSpaceLabel]; !ok {
		klog.Infof("Skipping Namespace %q as it does not contain the %q label", nsName.Name, dataSpaceLabel)
		return ctrl.Result{}, nil
	} else if v != "true" {
		klog.Infof("Skipping Namespace %q as it is not enabled for data spaces", nsName.Name)
		// Delete relevant NetworkPolicy and ConfigMap if found
		if err := r.deleteNetworkPolicy(ctx, npNsName); err != nil {
			return ctrl.Result{}, err
		}
		if err := r.deleteConfigMap(ctx, cmNsName); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create NetworkPolicy
	networkPolicy := forgeNetworkPolicy(nsName.Name)
	if err := r.createNetworkPolicy(ctx, nsName.Name, networkPolicy); err != nil {
		return ctrl.Result{}, err
	}

	// Create ConfigMap
	configMap, err := forgeConfigMap(nsName.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := r.createConfigMap(ctx, nsName.Name, configMap); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) deleteNetworkPolicy(ctx context.Context, nsName types.NamespacedName) error {
	var networkPolicy netv1.NetworkPolicy
	if err := r.Client.Get(ctx, nsName, &networkPolicy); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found NetworkPolicy %q in namespace %q", networkPolicyName, nsName.Namespace)
		} else {
			klog.Errorf("Error while getting NetworkPolicy %q in namespace %q", networkPolicyName, nsName.Namespace)
		}
		return err
	}

	r.Client.Delete(ctx, &networkPolicy)
	klog.Infof("Deleted NetworkPolicy %q in namespace %q", networkPolicyName, nsName.Namespace)
	return nil
}

func (r *NamespaceReconciler) deleteConfigMap(ctx context.Context, nsName types.NamespacedName) error {
	var configMap corev1.ConfigMap
	if err := r.Client.Get(ctx, nsName, &configMap); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found ConfigMap %q in namespace %q", configMapName, nsName.Namespace)
		} else {
			klog.Errorf("Error while getting ConfigMap %q in namespace %q", configMapName, nsName.Namespace)
		}
		return err
	}

	r.Client.Delete(ctx, &configMap)
	klog.Infof("Deleted ConfigMap %q in namespace %q", configMapName, nsName.Namespace)
	return nil
}

func (r *NamespaceReconciler) createNetworkPolicy(ctx context.Context, namespaceName string, networkPolicy *netv1.NetworkPolicy) error {
	if err := r.Client.Create(ctx, networkPolicy); err != nil {
		err = client.IgnoreAlreadyExists(err)
		if err == nil {
			klog.Infof("NetworkPolicy %q already exists in namespace %q", networkPolicyName, namespaceName)
		} else {
			klog.Errorf("Error while creating NetworkPolicy %q in namespace %q", networkPolicyName, namespaceName)
		}
		return err
	}
	klog.Infof("Created NetworkPolicy %q in namespace %q", networkPolicyName, namespaceName)
	return nil
}

func (r *NamespaceReconciler) createConfigMap(ctx context.Context, namespaceName string, configMap *corev1.ConfigMap) error {
	if err := r.Client.Create(ctx, configMap); err != nil {
		err = client.IgnoreAlreadyExists(err)
		if err == nil {
			klog.Infof("ConfigMap %q already exists in namespace %q", configMapName, namespaceName)
		} else {
			klog.Errorf("Error while creating ConfigMap %q in namespace %q", configMapName, namespaceName)
		}
		return err
	}
	klog.Infof("Created ConfigMap %q in namespace %q", configMapName, namespaceName)
	return nil
}

func forgeNetworkPolicy(namespaceName string) *netv1.NetworkPolicy {
	return &netv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkPolicyName,
			Namespace: namespaceName,
		},
		Spec: netv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					dataSpaceLabel: "true",
				},
			},
			PolicyTypes: []netv1.PolicyType{
				netv1.PolicyTypeIngress,
				netv1.PolicyTypeEgress,
			},
			Ingress: []netv1.NetworkPolicyIngressRule{{
				From: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							dataSpaceLabel:            "true", // For current namespace
							dataSpaceDestinationLabel: "true", // For other namespaces
						},
					},
				}},
			}},
			Egress: []netv1.NetworkPolicyEgressRule{{
				To: []netv1.NetworkPolicyPeer{{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							dataSpaceLabel:            "true", // For current namespace
							dataSpaceDestinationLabel: "true", // For other namespaces
						},
					},
				}},
			}},
		},
	}
}

func forgeConfigMap(namespaceName string) (*corev1.ConfigMap, error) {
	envoyConfig := forgeEnvoyConfig()

	marshaledEnvoyConfig, err := yaml.Marshal(envoyConfig)
	if err != nil {
		klog.Fatal("Could not marshal yaml, error: %v", err)
		return nil, err
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: namespaceName,
		},
		Data: map[string]string{
			"keys": string(marshaledEnvoyConfig),
		},
	}, nil
}

func forgeEnvoyConfig() *EnvoyConfig {
	return &EnvoyConfig{
		Admin: Admin{
			Address: Address{
				SocketAddress: SocketAddress{
					Address:   LOCALHOST_ADDR,
					PortValue: adminPort,
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
							Address:   LOCALHOST_ADDR,
							PortValue: egressTcpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: TcpProxyTypeName,
									TypedConfig: TypedConfig{
										Type:       TcpProxyTypeUrl,
										StatPrefix: egressTcpStatPrefixName,
										Cluster:    egressClusterName,
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
					ListenerFilters: []NameAndConfig{
						{
							Name: OriginalDstListenerFilterTypeName,
							TypedConfig: TypedConfig{
								Type: OriginalDstListenerFilterTypeUrl,
							},
						},
					},
				},
				// Ingress TCP listener
				{
					Name: "ingress_tcp_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   ANY_ADDR,
							PortValue: ingressTcpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: TcpProxyTypeName,
									TypedConfig: TypedConfig{
										Type:       TcpProxyTypeUrl,
										StatPrefix: ingressTcpStatPrefixName,
										Cluster:    ingressClusterName,
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: AccessLogTypeUrl,
											},
										}},
									},
								},
							},
						},
					},
					ListenerFilters: []NameAndConfig{
						{
							Name: OriginalDstListenerFilterTypeName,
							TypedConfig: TypedConfig{
								Type: OriginalDstListenerFilterTypeUrl,
							},
						},
					},
				},
				// Egress HTTP listener
				{
					Name: "egress_http_listener",
					Address: Address{
						SocketAddress: SocketAddress{
							Address:   LOCALHOST_ADDR,
							PortValue: egressHttpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: HttpConnectionManagerTypeName,
									TypedConfig: TypedConfig{
										Type:       HttpConnectionManagerTypeUrl,
										StatPrefix: egressHttpStatPrefixName,
										HttpFilters: []NameAndConfig{
											// Forward
											{
												Name: DynamicForwardProxyFilterTypeName,
												TypedConfig: TypedConfig{
													Type: DynamicForwardProxyFilterTypeUrl,
													DnsCacheConfig: DnsCacheConfig{
														Name:            dnsCacheConfigName,
														DnsLookupFamily: Ipv4Only,
													},
												},
											},
											// Router
											{
												Name: RouterTypeName,
												TypedConfig: TypedConfig{
													Type: RouterTypeUrl,
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
															Name: "egress_forward_rewrite_route",
															Match: Match{
																Prefix: "/host-rewrite",
															},
															Route: Route{
																Cluster: egressForwardClusterName,
															},
															TypedPerFilterConfig: TypedPerFilterConfig{
																DynamicForwardProxyType: DynamicForwardProxyType{
																	Type:               DynamicForwardProxyRouteTypeUrl,
																	HostRewriteLiteral: "dst-svc.dst-space.svc.cluster.local",
																},
															},
														},
														{
															Name: "egress_forward_route",
															Match: Match{
																Prefix: "/",
															},
															Route: Route{
																Cluster: egressForwardClusterName,
															},
														},
													},
												},
											},
										},
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: AccessLogTypeUrl,
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
							Address:   ANY_ADDR,
							PortValue: ingressHttpPort,
						},
					},
					FilterChains: []FilterChain{
						{
							Filters: []NameAndConfig{
								{
									Name: HttpConnectionManagerTypeName,
									TypedConfig: TypedConfig{
										Type:       HttpConnectionManagerTypeUrl,
										StatPrefix: ingressHttpStatPrefixName,
										HttpFilters: []NameAndConfig{
											// Forward
											{
												Name: DynamicForwardProxyFilterTypeName,
												TypedConfig: TypedConfig{
													Type: DynamicForwardProxyFilterTypeUrl,
													DnsCacheConfig: DnsCacheConfig{
														Name:            dnsCacheConfigName,
														DnsLookupFamily: Ipv4Only,
													},
												},
											},
											// Router
											{
												Name: RouterTypeName,
												TypedConfig: TypedConfig{
													Type: RouterTypeUrl,
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
																Cluster: ingressForwardClusterName,
															},
														},
													},
												},
											},
										},
										// Log access to /dev/stdout
										AccessLog: []NameAndConfig{{
											Name: AccessLogTypeName,
											TypedConfig: TypedConfig{
												Type: AccessLogTypeUrl,
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
					Name:            egressClusterName,
					DnsLookupFamily: Ipv4Only,
					Type:            OriginalDstType,
					LbPolicy:        OriginalDstLbPolicy,
					ConnectTimeout:  "6s",
					OriginalDstLbConfig: OriginalDstLbConfig{
						UseHttpHeader: true,
					},
				},
				// Ingress cluster
				{
					Name:            ingressClusterName,
					DnsLookupFamily: Ipv4Only,
					Type:            OriginalDstType,
					LbPolicy:        OriginalDstLbPolicy,
					ConnectTimeout:  "6s",
					OriginalDstLbConfig: OriginalDstLbConfig{
						UseHttpHeader: true,
					},
				},
				// Egress forward cluster
				{
					Name:           egressForwardClusterName,
					LbPolicy:       OriginalDstLbPolicy,
					ConnectTimeout: "6s",
					ClusterType: NameAndConfig{
						Name: DynamicForwardProxyClusterTypeName,
						TypedConfig: TypedConfig{
							Type: DynamicForwardProxyClusterTypeUrl,
							DnsCacheConfig: DnsCacheConfig{
								Name:            dnsCacheConfigName,
								DnsLookupFamily: Ipv4Only,
							},
						},
					},
				},
				// Ingress forward cluster
				{
					Name:           ingressForwardClusterName,
					LbPolicy:       OriginalDstLbPolicy,
					ConnectTimeout: "6s",
					ClusterType: NameAndConfig{
						Name: DynamicForwardProxyClusterTypeName,
						TypedConfig: TypedConfig{
							Type: DynamicForwardProxyClusterTypeUrl,
							DnsCacheConfig: DnsCacheConfig{
								Name:            dnsCacheConfigName,
								DnsLookupFamily: Ipv4Only,
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
