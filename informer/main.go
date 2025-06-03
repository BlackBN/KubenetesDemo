package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {
	// config, err := clientcmd.BuildConfigFromFlags("", "/Users/bn/.kube/waf-saas-config")
	// if err != nil {
	// 	panic(err)
	// }
	// clientSet, err := kubernetes.NewForConfig(config)
	// if err != nil {
	// 	panic(err)
	// }
	var err error
	var config *rest.Config
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "waf-saas-config"), "")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "")
	}
	if config, err = rest.InClusterConfig(); err != nil {
		if config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig); err != nil {
			panic(err.Error())
		}
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	stopCh := make(chan struct{})
	defer close(stopCh)

	sharedInformer := informers.NewSharedInformerFactory(clientSet, 30*time.Second)
	deploymentInformer := sharedInformer.Apps().V1().Deployments()
	lister := deploymentInformer.Lister()
	informer := deploymentInformer.Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			mObj := obj.(*v1.Deployment)
			fmt.Printf("add deploy to store: %s\n", mObj.GetName())
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			old := oldObj.(*v1.Deployment)
			new := newObj.(*v1.Deployment)
			fmt.Printf("update info : new deploy name %s, old deploy name %s\n", new.Name, old.Name)
		},
		DeleteFunc: func(obj interface{}) {
			mObj := obj.(*v1.Deployment)
			fmt.Printf("delete deploy from store:%s\n", mObj.Name)
		},
	})

	sharedInformer.Start(stopCh)
	sharedInformer.WaitForCacheSync(stopCh)
	deployments, err := lister.Deployments("waf").List(labels.Everything())
	if err != nil {
		panic(err)
	}
	for _, d := range deployments {
		fmt.Printf("deployment info %s/%s\n", d.Namespace, d.Name)
	}
	<-stopCh
}
