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

	"data-space.liqo.io/common"
	"data-space.liqo.io/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	nsName := req.NamespacedName
	klog.Infof("Reconcile Pod %q", nsName.Name)

	pod := &corev1.Pod{}
	if err := r.Get(ctx, nsName, pod); err != nil {
		err = client.IgnoreNotFound(err)
		if err == nil {
			klog.Infof("Skipping not found Pod %q", nsName.Name)
		}
		return ctrl.Result{}, err
	}

	podInitialized := false
	for _, value := range pod.Status.Conditions {
		if value.Type == corev1.PodInitialized && value.Status == corev1.ConditionTrue {
			podInitialized = true
			break
		}
	}
	if !podInitialized {
		klog.Infof("Skipping Pod %q as it is not yet initialized", nsName.Name)
		return ctrl.Result{}, nil
	}

	if value, ok := pod.ObjectMeta.Labels[consts.DataSpaceLabel]; ok && value == "true" {
		klog.Infof("Skipping Pod %q as it already has label %q", fmt.Sprintf("%s: %s", consts.DataSpaceLabel, "true"), nsName.Name)
		return ctrl.Result{}, nil
	}

	common.InjectPodLabel(pod, consts.DataSpaceLabel, "true")
	if err := r.Client.Status().Update(ctx, pod); err != nil {
		klog.Errorf("Error while updating Pod %q status in namespace %q", consts.NetworkPolicyName, nsName.Namespace)
	}
	klog.Infof("Added label %q to Pod %q in namespace %q", fmt.Sprintf("%s: %s", consts.DataSpaceLabel, "true"), nsName.Name, nsName.Namespace)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	noDeleteEventPredicate := predicate.Funcs{
		DeleteFunc: func(de event.DeleteEvent) bool {
			return false
		},
	}
	mutatedPodPredicate, err := predicate.LabelSelectorPredicate(metav1.LabelSelector{
		MatchLabels: map[string]string{
			consts.MutatedPodLabel: "true",
		},
	})
	if err != nil {
		klog.Error(err)
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(noDeleteEventPredicate).
		For(&corev1.Pod{}, builder.WithPredicates(mutatedPodPredicate)).
		Complete(r)
}
