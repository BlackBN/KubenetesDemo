package main

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func NamespaceIndexFunc(obj interface{}) (result []string, err error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("type is error %v", err)
	}
	result = []string{pod.Namespace}
	return
}

func NodeNameIndexFunc(obj interface{}) (result []string, err error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("type is error %v", err)
	}
	result = []string{pod.Spec.NodeName}
	return
}
func main() {

	index := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		"namespace1": NamespaceIndexFunc,
		"nodeName1":  NodeNameIndexFunc,
	})

	pod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "index-pod-1",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
		},
	}
	pod2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "index-pod-2",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			NodeName: "node2",
		},
	}
	pod3 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "index-pod-3",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			NodeName: "node1",
		},
	}

	index.Add(pod1)
	index.Add(pod2)
	index.Add(pod3)
	pods, err := index.ByIndex("namespace1", "kube-system")
	if err != nil {
		panic(err)
	}
	for _, s := range pods {
		fmt.Println(s.(*corev1.Pod).Name)
	}
	fmt.Println("----------")
	pods, err = index.ByIndex("nodeName1", "node1")
	if err != nil {
		panic(err)
	}
	for _, s := range pods {
		fmt.Println(s.(*corev1.Pod).Name)
	}
}
