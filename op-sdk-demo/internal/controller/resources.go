package controller

import (
	demov1beta1 "github.com/BlackBN/KubenetesDemo/op-sdk-demo/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewDeploy(as *demov1beta1.AppService) *appsv1.Deployment {
	labels := map[string]string{"myappservice": as.Name}
	selector := v1.LabelSelector{
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		TypeMeta: v1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            as.Name,
			Namespace:       as.Namespace,
			OwnerReferences: makeOwnerReferences(as),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: as.Spec.Size,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: newContainers(as),
				},
			},
			Selector: &selector,
		},
	}
}
func newContainers(as *demov1beta1.AppService) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range as.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}
	return []corev1.Container{
		{
			Name:      "c1",
			Image:     as.Spec.Image,
			Resources: as.Spec.Resources,
			Env:       as.Spec.Envs,
			Ports:     containerPorts,
		},
	}
}

func makeOwnerReferences(as *demov1beta1.AppService) []v1.OwnerReference {
	return []v1.OwnerReference{
		*v1.NewControllerRef(as, schema.GroupVersionKind{
			Kind:    demov1beta1.Kind,
			Version: demov1beta1.GroupVersion.Version,
			Group:   demov1beta1.GroupVersion.Group,
		}),
	}
}

func NewService(as *demov1beta1.AppService) *corev1.Service {
	return &corev1.Service{
		TypeMeta: v1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: v1.ObjectMeta{
			Name:            as.Name,
			Namespace:       as.Namespace,
			OwnerReferences: makeOwnerReferences(as),
		},
		Spec: corev1.ServiceSpec{
			Ports: as.Spec.Ports,
			Type:  corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"myappservice": as.Name,
			},
		},
	}
}
