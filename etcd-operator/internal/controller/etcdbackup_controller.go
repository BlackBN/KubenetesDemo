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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "github.com/BlackBN/KubenetesDemo/etcd-operator/api/v1alpha1"
)

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	Recorder    record.EventRecorder
	BackupImage string
}

type backupState struct {
	backup  *etcdv1alpha1.EtcdBackup
	actual  *backupStateContainer
	desired *backupStateContainer
}

type backupStateContainer struct {
	pod *corev1.Pod
}

func (r *EtcdBackupReconciler) getState(ctx context.Context, req ctrl.Request) (*backupState, error) {
	var state backupState
	state.backup = &etcdv1alpha1.EtcdBackup{}
	if err := r.Get(ctx, req.NamespacedName, state.backup); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, fmt.Errorf("getting backup object error : %v", err)
		}
		state.backup = nil
		return &state, nil
	}
	if err := r.setStateActual(ctx, &state); err != nil {
		return nil, fmt.Errorf("setting actual state err : %v", err)
	}
	if err := r.setStateDesired(ctx, &state); err != nil {
		return nil, fmt.Errorf("setting desired state err : %v", err)
	}
	return &state, nil
}

func (r *EtcdBackupReconciler) setStateDesired(ctx context.Context, state *backupState) error {
	var desired backupStateContainer
	pod, err := r.PodForBackup(state.backup)
	if err != nil {
		return nil
	}
	if err := controllerutil.SetControllerReference(state.backup, pod, r.Scheme); err != nil {
		return fmt.Errorf("set controller reference error : %v", err)
	}
	desired.pod = pod
	state.desired = &desired
	return nil
}

func (r *EtcdBackupReconciler) PodForBackup(backup *etcdv1alpha1.EtcdBackup) (*corev1.Pod, error) {

	return &corev1.Pod{
		ObjectMeta: v1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "c1",
					Image: r.BackupImage,
					Args: []string{

						//可以写成模板化
						"--etcd-url", backup.Spec.EtcdUrl,
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ENDPOINT",
							Value: backup.Spec.Source.Endpoint,
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: backup.Spec.Source.Secret,
								},
							},
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}, nil
}

func (r *EtcdBackupReconciler) setStateActual(ctx context.Context, state *backupState) error {
	var actual backupStateContainer
	key := client.ObjectKey{
		Name:      state.backup.Name,
		Namespace: state.backup.Namespace,
	}
	actual.pod = &corev1.Pod{}
	if err := r.Get(ctx, key, actual.pod); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("getting pod error : %v", err)
		}
		actual.pod = nil
	}
	state.actual = &actual
	return nil
}

// +kubebuilder:rbac:groups=etcd.bnblak.com,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=etcd.bnblak.com,resources=etcdbackups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=etcd.bnblak.com,resources=etcdbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	state, err := r.getState(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}
	var action Action
	switch {
	case state.backup == nil: //被删除了
		log.Info("backup object not found")
	case !state.backup.DeletionTimestamp.IsZero():
		log.Info("backup object has been delete")
	case state.backup.Status.Phase == "":
		log.Info("backup starting......")
		r.Recorder.Event(state.backup, corev1.EventTypeNormal, "SuccessfulStarting", fmt.Sprintf(""))
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseBackingUp
		action = &PatchStatus{
			client: r.Client,
			oldObj: state.backup,
			newObj: newBackup,
		}
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBackupPhaseFailed:
		log.Info("backup has failed, Ignoring....")
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBackupPhaseCompleted:
		log.Info("backup has completed, Ignoring....")
	case state.actual.pod == nil:
		log.Info("backup actual pod is nil,need create...")
		r.Recorder.Event(state.backup, corev1.EventTypeNormal, "SuccessfulCreate", fmt.Sprintf(""))
		action = &CreateObject{
			client: r.Client,
			obj:    state.desired.pod,
		}
	case state.actual.pod.Status.Phase == corev1.PodFailed:
		log.Info("backup actual pod status phase is failed")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseFailed
		action = &PatchStatus{
			client: r.Client,
			oldObj: state.backup,
			newObj: newBackup,
		}
		r.Recorder.Event(state.backup, corev1.EventTypeWarning, "BackupFailed", fmt.Sprintf(""))

	case state.actual.pod.Status.Phase == corev1.PodSucceeded:
		log.Info("backup actual pod status phase is succeed")
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseCompleted
		action = &PatchStatus{
			client: r.Client,
			oldObj: state.backup,
			newObj: newBackup,
		}
		r.Recorder.Event(state.backup, corev1.EventTypeNormal, "BuckupSuccess", fmt.Sprintf(""))

	}

	if action != nil {
		if err := action.Execute(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdBackup{}).
		Named("etcdbackup").
		Owns(&corev1.Pod{}).
		Complete(r)
}
