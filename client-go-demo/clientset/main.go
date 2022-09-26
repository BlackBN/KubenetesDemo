package main

import (
	"context"
	"fmt"

	"k8s.io/client-go/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", "/Users/bn/.kube/ry_controller")
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	deploymentList, err := clientset.AppsV1().Deployments("kube-system").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		panic(err)
	}
	for _, deploy := range deploymentList.Items {
		fmt.Println(deploy.Name)

	}
}
