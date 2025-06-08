package controller

import (
	"strconv"

	etcdv1alpha1 "github.com/BlackBN/KubenetesDemo/etcd-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EtcdClusterLabelsName   = "etcd.bnblak.com/name"
	EtcdClusterLabelsCommon = "app"

	EtcdClusterLabelsCommonValue = "etcd"

	EtcdDataDirName = "datadir"
)

func MutateHeadlessService(ec *etcdv1alpha1.EtcdCluster, svc *corev1.Service) {
	svc.Labels = map[string]string{
		EtcdClusterLabelsCommon: EtcdClusterLabelsCommonValue,
	}
	svc.Spec = corev1.ServiceSpec{
		ClusterIP: corev1.ClusterIPNone,
		Ports: []corev1.ServicePort{
			{
				Name: "peer",
				Port: 2380,
			},
			{
				Name: "client",
				Port: 2379,
			},
		},
		Type: corev1.ServiceTypeNodePort,
		Selector: map[string]string{
			EtcdClusterLabelsName: ec.Name,
		},
	}
}

func MutateStatefulSet(ec *etcdv1alpha1.EtcdCluster, sts *appsv1.StatefulSet) {
	sts.Labels = map[string]string{
		EtcdClusterLabelsCommon: EtcdClusterLabelsCommonValue,
	}
	sts.Spec = appsv1.StatefulSetSpec{
		Replicas:    ec.Spec.Size,
		ServiceName: ec.Name,
		Selector: &v1.LabelSelector{
			MatchLabels: map[string]string{
				EtcdClusterLabelsName: ec.Name,
			},
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: map[string]string{
					EtcdClusterLabelsCommon: EtcdClusterLabelsCommonValue,
					EtcdClusterLabelsName:   ec.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(ec),
				ImagePullSecrets: []corev1.LocalObjectReference{
					{
						Name: "talentsec-registry-secret",
					},
				},
			},
		},
		VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
			{
				ObjectMeta: v1.ObjectMeta{
					Name: EtcdDataDirName,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			},
		},
	}

}

func newContainers(ec *etcdv1alpha1.EtcdCluster) []corev1.Container {
	return []corev1.Container{
		{
			Name:            ec.Name,
			Image:           ec.Spec.Image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 2380,
					Protocol:      corev1.ProtocolTCP,
					Name:          "peer",
				},
				{
					ContainerPort: 2379,
					Protocol:      corev1.ProtocolTCP,
					Name:          "client",
				},
			},
			Env: []corev1.EnvVar{
				{
					Name:  "INITIAL_CLUSTER_SIZE",
					Value: strconv.Itoa(int(*ec.Spec.Size)),
				},
				{
					Name:  "SET_NAME",
					Value: ec.Name,
				},
				{
					Name: "MY_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
				{
					Name: "POD_IP",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "status.podIP",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      EtcdDataDirName,
					MountPath: "/var/run/etcd",
				},
			},
			Command: []string{
				"/bin/sh", "-ec",
				"",
			},
			Lifecycle: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"/bin/sh", "-ec",
							"",
						},
					},
				},
			},
		},
	}
}
