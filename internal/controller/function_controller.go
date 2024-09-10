package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	k8sfunctionsv1alpha1 "github.com/pdesai-dev/k8s-function/api/v1alpha1"
)

// FunctionReconciler reconciles a Function object
type FunctionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=k8s-function.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=k8s-function.io,resources=functions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=k8s-function.io,resources=functions/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var function k8sfunctionsv1alpha1.Function
	if err := r.Get(ctx, req.NamespacedName, &function); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Function resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Function")
		return ctrl.Result{}, err
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList,
		client.InNamespace(req.Namespace),
		client.MatchingLabels(labelsForFunction(&function))); err != nil {
		logger.Error(err, "Unable to list child Pods")
		return ctrl.Result{}, err
	}

	activePods := make([]corev1.Pod, 0)
	completedPods := make([]corev1.Pod, 0)

	for _, pod := range podList.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning, corev1.PodPending:
			activePods = append(activePods, pod)
		case corev1.PodSucceeded, corev1.PodFailed:
			completedPods = append(completedPods, pod)
		}
	}

	logger.Info("Pod counts", "active", len(activePods), "desired", *function.Spec.Replicas, "completed", len(completedPods))

	sort.Slice(activePods, func(i, j int) bool {
		if activePods[i].Status.Phase != activePods[j].Status.Phase {
			return activePods[i].Status.Phase == corev1.PodPending
		}
		return activePods[i].CreationTimestamp.Before(&activePods[j].CreationTimestamp)
	})

	if len(activePods) < int(*function.Spec.Replicas) {
		podsToCreate := int(*function.Spec.Replicas) - len(activePods) 
		for i := 0; i < podsToCreate; i++ {
			pod := r.podForFunction(&function)
			logger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			if err := r.Create(ctx, pod); err != nil {
				logger.Error(err, "Failed to create new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
				return ctrl.Result{}, err
			}
		}
	} else if len(activePods) > int(*function.Spec.Replicas) {
		podsToDelete := activePods[*function.Spec.Replicas:]
		for _, pod := range podsToDelete {
			logger.Info("Deleting active Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
			if err := r.Delete(ctx, &pod); err != nil {
				logger.Error(err, "Failed to delete active Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
				return ctrl.Result{}, err
			}
		}
	} else {
		for _, pod := range completedPods {
			completionTime := pod.CreationTimestamp.Time
			age := time.Since(completionTime)
			if age > time.Duration(*function.Spec.TTLSecondsAfterFinished)*time.Second {
				logger.Info("Deleting completed Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
				if err := r.Delete(ctx, &pod); err != nil {
					logger.Error(err, "Failed to delete completed Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
					return ctrl.Result{}, err
				}
			}
		}
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := r.Get(ctx, req.NamespacedName, &function); err != nil {
			return err
		}
		function.Status.Replicas = int32(len(activePods))
		function.Status.Active = int32(len(activePods))
		function.Status.Completed = int32(len(completedPods))
		function.Status.Selector = labels.Set(labelsForFunction(&function)).String()
		return r.Status().Update(ctx, &function)
	})

	if err != nil {
		logger.Error(err, "Failed to update Function status")
		return ctrl.Result{}, err
	}

	// Requeue after 5 minutes so we can do some cleanup
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *FunctionReconciler) podForFunction(function *k8sfunctionsv1alpha1.Function) *corev1.Pod {
	labels := labelsForFunction(function)
	envVars := []corev1.EnvVar{}
  	for key, value := range function.Spec.EnvVariables {
		envVars = append(envVars, corev1.EnvVar{
		Name:  key,
		Value: value,
		})
	}
	command := []string{}
	command = append(command, "python", "-u", "-c", function.Spec.Code)
	command = append(command, function.Spec.Args...)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-function-%s", function.Name, rand.String(5)),
			Namespace: function.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:  "function",
				Image: function.Spec.RuntimeImage,
				Command: command,
				/*
				Command: []string{
					"python", 
					"-u",
					"-c",
					function.Spec.Code,
				  },
				  */
				  Env: envVars,
			}},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
	ctrl.SetControllerReference(function, pod, r.Scheme)
	return pod
}

func labelsForFunction(function *k8sfunctionsv1alpha1.Function) map[string]string {
	return map[string]string{"app": "function", "function": function.Name}
}

func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&k8sfunctionsv1alpha1.Function{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}