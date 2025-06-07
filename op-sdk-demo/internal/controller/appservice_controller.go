/*
Copyright 2025 bn.

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
	"encoding/json"
	"reflect"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	demov1beta1 "github.com/BlackBN/KubenetesDemo/op-sdk-demo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old-app-spec"
)

// AppServiceReconciler reconciles a AppService object
type AppServiceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=demo.bnblak.com,resources=appservices,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=demo.bnblak.com,resources=appservices/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=demo.bnblak.com,resources=appservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AppService object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile
func (r *AppServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	log.Info("begin Reconcile")
	// TODO(user): your logic here
	var myAppService demov1beta1.AppService
	err := r.Client.Get(ctx, req.NamespacedName, &myAppService)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}
	// 当前对象标记为了删除
	if myAppService.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	deploy := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, req.NamespacedName, deploy); err != nil && apierrors.IsNotFound(err) {
		// 关联 anno
		data, err := json.Marshal(myAppService.Spec)
		if err != nil {
			return ctrl.Result{}, err
		}
		if myAppService.Annotations != nil {
			myAppService.Annotations[oldSpecAnnotation] = string(data)
		} else {
			myAppService.Annotations = map[string]string{
				oldSpecAnnotation: string(data),
			}
		}
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return r.Client.Update(ctx, &myAppService)
		}); err != nil {
			return ctrl.Result{}, err
		}
		newDeploy := NewDeploy(&myAppService)
		if err := r.Client.Create(ctx, newDeploy); err != nil {
			return ctrl.Result{}, err
		}
		newService := NewService(&myAppService)
		if err := r.Client.Create(ctx, newService); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}
	oldSpec := demov1beta1.AppServiceSpec{}
	if err := json.Unmarshal([]byte(myAppService.Annotations[oldSpecAnnotation]), &oldSpec); err != nil {
		return ctrl.Result{}, err
	}
	if !reflect.DeepEqual(myAppService.Spec, oldSpec) {
		oldDeploy := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, req.NamespacedName, oldDeploy); err != nil {
			return ctrl.Result{}, err
		}
		oldDeploy.Spec = NewDeploy(&myAppService).Spec
		// 一般不会直接调用 Update 方法进行更新
		// if err := r.Client.Update(ctx, oldDeploy); err != nil {
		// 	return ctrl.Result{}, err
		// }
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return r.Client.Update(ctx, oldDeploy)
		}); err != nil {
			return ctrl.Result{}, err
		}

		oldService := &corev1.Service{}
		if err := r.Client.Get(ctx, req.NamespacedName, oldService); err != nil {
			return ctrl.Result{}, err
		}

		oldService.Spec = NewService(&myAppService).Spec
		// 一般不会直接调用 Update 方法进行更新
		// if err := r.Client.Update(ctx, oldDeploy); err != nil {
		// 	return ctrl.Result{}, err
		// }
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			return r.Client.Update(ctx, oldService)
		}); err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AppServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&demov1beta1.AppService{}).
		Complete(r)
}
