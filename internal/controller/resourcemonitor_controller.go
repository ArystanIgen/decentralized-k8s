/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	resourcev1 "github.com/ArystanIgen/decentralized-k8s/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/examples/client-go/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ResourceMonitorReconciler reconciles a ResourceMonitor object
type ResourceMonitorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	MetricsClient *versioned.Clientset
	KubeClient    *kubernetes.Clientset
}

type ResourceUtilization struct {
	CPU    float64
	Memory float64
}

//+kubebuilder:rbac:groups=core.example.com,resources=resourcemonitors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.example.com,resources=resourcemonitors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.example.com,resources=resourcemonitors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ResourceMonitor object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *ResourceMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var resourceMonitor resourcev1.ResourceMonitor
	if err := r.Get(ctx, req.NamespacedName, &resourceMonitor); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the node where this pod is running
	pod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, pod); err != nil {
		logger.Error(err, "unable to fetch Pod")
		return ctrl.Result{}, err
	}
	nodeName := pod.Spec.NodeName

	logger.Info(nodeName)

	if nodeName != "" {
		err := r.checkAndAdjustNodeResources(ctx, nodeName, &resourceMonitor)
		if err != nil {
			logger.Error(err, "failed to adjust node resources", "node", nodeName)
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ResourceMonitorReconciler) checkAndAdjustNodeResources(ctx context.Context, nodeName string, resourceMonitor *resourcev1.ResourceMonitor) error {
	utilization, err := r.getNodeResourceUtilization(ctx, nodeName)
	if err != nil {
		return fmt.Errorf("error getting resource utilization for node %s: %v", nodeName, err)
	}

	fmt.Printf("Current Utilization on Node %s: CPU: %f%%, Memory: %f%%\n", nodeName, utilization.CPU, utilization.Memory)

	if utilization.CPU > float64(resourceMonitor.Spec.CPUThresholdPercent) {
		fmt.Printf("CPU threshold exceeded on node %s; consider scaling up CPU resources or reducing workload.\n", nodeName)
	}

	if utilization.Memory > float64(resourceMonitor.Spec.MemoryThresholdPercent) {
		fmt.Printf("Memory threshold exceeded on node %s; consider scaling up memory resources or reducing workload.\n", nodeName)
	}

	return nil
}

func (r *ResourceMonitorReconciler) getNodeResourceUtilization(ctx context.Context, nodeName string) (ResourceUtilization, error) {
	return ResourceUtilization{CPU: 75, Memory: 80}, nil // Dummy data for illustration
}

func (r *ResourceMonitorReconciler) adjustNodeResources(ctx context.Context, nodeName string) error {
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1.ResourceMonitor{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
