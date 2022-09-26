package main

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"k8s.io/client-go/util/homedir"
)

var kubeconfig *string

func main() {
	if home := homedir.HomeDir(); home != "" {
		fmt.Printf("home : %s\n", home)
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "ry_controller"), "absolute path to the ry k8s config")
	} else {
		kubeconfig = flag.String("kubeconfig", filepath.Join("Users", "bn", ".kube", "ry_controller"), "absolute path to the ry k8s config")
	}

	flag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}
	config.APIPath = "api"
	config.GroupVersion = &corev1.SchemeGroupVersion
	config.NegotiatedSerializer = scheme.Codecs
	restClient, err := rest.RESTClientFor(config)

	if err != nil {
		panic(err.Error())
	}

	result := &corev1.PodList{}

	namespace := "kube-system"

	err = restClient.Get().
		Namespace(namespace).
		Resource("pods").
		VersionedParams(&metav1.ListOptions{}, scheme.ParameterCodec).
		Do(context.TODO()).
		Into(result)

	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("namespace\t status\t\t name\n")

	for _, d := range result.Items {
		fmt.Printf("%v\t %v\t %v\n",
			d.Namespace,
			d.Status.Phase,
			d.Name)
	}

}
