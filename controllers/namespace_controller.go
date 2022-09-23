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

const (
	dataSpaceLabel            = "data-space/enabled"
	dataSpaceDestinationLabel = "data-space-dest/enabled"
	networkPolicyName         = "data-space-network-policy"
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

	namespace := corev1.Namespace{}
	if err := r.Get(ctx, nsName, &namespace); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found Namespace %q", nsName.Name)
			// Delete relevant Network Policy if found
			if err := r.deleteNetworkPolicy(ctx, npNsName); err != nil {
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
		// Delete associated Network Policy if found
		if err := r.deleteNetworkPolicy(ctx, npNsName); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create NetworkPolicy
	networkPolicy := forgeNetworkPolicy(nsName.Name)
	if err := r.createNetworkPolicy(ctx, nsName.Name, networkPolicy); err != nil {
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

func (r *NamespaceReconciler) createNetworkPolicy(ctx context.Context, namespaceName string, networkPolicy *netv1.NetworkPolicy) error {
	if err := r.Client.Create(ctx, networkPolicy); err != nil {
		err = client.IgnoreAlreadyExists(err)
		if err == nil {
			klog.Infof("NetworkPolicy %q already exists in namespace %q", networkPolicyName, namespaceName)
		} else {
			klog.Errorf("Error while creating NetworkPolicy %q in Namespace %q", networkPolicyName, namespaceName)
		}
		return err
	}
	klog.Infof("Created NetworkPolicy %q in Namespace %q", networkPolicyName, namespaceName)
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

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
