/*
Copyright 2022 Ozan YILDIZ.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	testv1alpha1 "github.com/yildizozan/sauron/api/v1alpha1"
)

// ApiTestReconciler reconciles a ApiTest object
type ApiTestReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=test.yildizozan.com,resources=apitests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=test.yildizozan.com,resources=apitests/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=test.yildizozan.com,resources=apitests/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ApiTest object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *ApiTestReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the Apitest instance
	apitest := &testv1alpha1.ApiTest{}
	err := r.Get(ctx, req.NamespacedName, apitest)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			logger.Info("Apitest resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get apitest")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{
		Namespace: apitest.Namespace,
		Name:      apitest.Name,
	}, found)
	if err != nil && errors.IsNotFound(err) {
		deployment := r.createDeployment(apitest)
		logger.Info("Creating a new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
		err = r.Create(ctx, deployment)
		if err != nil {
			logger.Error(err, "Failed to create new Deployment", "Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		logger.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 10}, nil
}

// deploymentForMemcached returns a memcached Deployment object
func (r *ApiTestReconciler) createDeployment(m *testv1alpha1.ApiTest) *appsv1.Deployment {
	ls := map[string]string{"app": "memcached", "memcached_cr": m.Name}

	containers := []corev1.Container{
		{
			Image:   "busybox:latest",
			Name:    "busybox",
			Command: []string{"/bin/sh", "-c", "while true; do date; sleep 1; done"},
			Ports: []corev1.ContainerPort{{
				ContainerPort: 8080,
				Name:          "busybox",
			}},
		},
	}

	podSpec := corev1.PodSpec{
		Containers: containers,
	}

	podTemplateSpec := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: ls,
		},
		Spec: podSpec,
	}

	deploymentSpec := appsv1.DeploymentSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Template: podTemplateSpec,
	}

	objectMeta := metav1.ObjectMeta{
		Name:      m.Name,
		Namespace: "default",
	}

	deployment := &appsv1.Deployment{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: objectMeta,
		Spec:       deploymentSpec,
		Status:     appsv1.DeploymentStatus{},
	}

	// Set Memcached instance as the owner and controller
	ctrl.SetControllerReference(m, deployment, r.Scheme)
	return deployment
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApiTestReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testv1alpha1.ApiTest{}).
		Complete(r)
}
