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
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/examples/client-go/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

// ResourceMonitorReconciler reconciles a ResourceMonitor object
type ResourceMonitorReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	MetricsClient *versioned.Clientset
	KubeClient    *kubernetes.Clientset
	Log           logr.Logger
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
	log := r.Log.WithValues("resourcemonitor", req.NamespacedName)

	var monitor resourcev1.ResourceMonitor
	if err := r.Get(ctx, req.NamespacedName, &monitor); err != nil {
		log.Error(err, "unable to fetch ResourceMonitor")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 1. Monitor node resource utilization
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		log.Error(err, "unable to list nodes")
		return ctrl.Result{}, err
	}

	for _, node := range nodeList.Items {
		nodeUtilization, err := getNodeResourceUtilization(ctx, node, r.Client)
		if err != nil {
			log.Error(err, "unable to get node resource utilization")
			continue
		}

		if nodeUtilization.CPU > float64(monitor.Spec.CPUThresholdPercent) || nodeUtilization.Memory > float64(monitor.Spec.MemoryThresholdPercent) {
			// 2. Adjust pod limits if thresholds are exceeded
			podList := &corev1.PodList{}
			labelSelector := labels.SelectorFromSet(labels.Set{"app": "resource-intensive"})
			listOpts := &client.ListOptions{Namespace: req.Namespace, LabelSelector: labelSelector}
			if err := r.List(ctx, podList, listOpts); err != nil {
				log.Error(err, "unable to list pods for adjustment")
				continue
			}

			for _, pod := range podList.Items {
				adjustPodResources(ctx, &pod, r.Client, log)
			}

			// 3. Optionally notify a central controller if significant adjustments are made
			notifyCentralController("Significant resource adjustment made for node: " + node.Name)
		}
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

func getNodeResourceUtilization(ctx context.Context, node corev1.Node, cl client.Client) (utilization ResourceUtilization, err error) {
	// Dummy function to fetch node resource utilization
	// In practice, you would probably use metrics-server or similar to get actual usage data
	return ResourceUtilization{CPU: 85, Memory: 90}, nil
}

func adjustPodResources(ctx context.Context, pod *corev1.Pod, cl client.Client, log logr.Logger) {
	// Adjust resources for the pod
	for i := range pod.Spec.Containers {
		pod.Spec.Containers[i].Resources.Limits["cpu"] = resource.MustParse("500m")   // Example adjustment
		pod.Spec.Containers[i].Resources.Limits["memory"] = resource.MustParse("1Gi") // Example adjustment
	}
	if err := cl.Update(ctx, pod); err != nil {
		log.Error(err, "failed to update pod resources", "pod", pod.Name)
	}
}

func notifyCentralController(message string) {
	// Implement actual communication logic, e.g., via REST API, message queue, etc.
	fmt.Println("Notification sent to central controller:", message)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ResourceMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&resourcev1.ResourceMonitor{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
