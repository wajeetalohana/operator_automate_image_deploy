/*
Copyright 2023.

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
	"reflect"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "example.com/m/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//logger
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Log    logr.Logger //TODO: Use logger
}

//+kubebuilder:rbac:groups=cache.my.domain,resources=projects,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.my.domain,resources=projects/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.my.domain,resources=projects/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete;apply;
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;apply

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Project object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// Get the Project custom resource
	var project cachev1alpha1.Project
	if err := r.Client.Get(ctx, req.NamespacedName, &project); err != nil {
		return ctrl.Result{}, err
	}
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name + "-deployment",
			Namespace: project.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": project.Name,
				},
			},
			Replicas: &project.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": project.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  project.Name,
							Image: project.Spec.Image,
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(&project, &deployment, r.Scheme); err != nil {
		// Handle the error
		return ctrl.Result{}, err
	}

	// Create the Deployment object
	foundDeployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		err = r.Create(ctx, &deployment)
		if err != nil {
			return ctrl.Result{}, err
		}

		// Deployment created successfully - return and requeue
		return ctrl.Result{}, nil
	} else if err != nil {
		//log.Error(err, "unable to get Deployment")
		return ctrl.Result{}, err
	}

	if !reflect.DeepEqual(deployment.Spec, foundDeployment.Spec) {
		foundDeployment.Spec = deployment.Spec
		err = r.Update(ctx, foundDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Project{}).
		Complete(r)
}
