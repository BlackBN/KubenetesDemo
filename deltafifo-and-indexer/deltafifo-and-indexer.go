package deltafifoandindexer

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

func Process(indexer cache.Store) func(obj interface{}, isInInitialList bool) error {
	return func(obj interface{}, isInInitialList bool) error {
		if deltas, ok := obj.(cache.Deltas); ok {
			for _, d := range deltas {
				obj := d.Object

				switch d.Type {
				case cache.Sync, cache.Replaced, cache.Added, cache.Updated:
					if _, exists, err := indexer.Get(obj); err == nil && exists {
						if err := indexer.Update(obj); err != nil {
							return err
						}
					} else {
						if err := indexer.Add(obj); err != nil {
							return err
						}
					}
				case cache.Deleted:
					if err := indexer.Delete(obj); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
}

func Execute() {

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{
		"namespace": NamespaceIndexFunc,
		"nodename":  NodeNameIndexFunc,
	})
	deltafifo := cache.NewDeltaFIFOWithOptions(cache.DeltaFIFOOptions{
		KeyFunction:  cache.MetaNamespaceKeyFunc,
		KnownObjects: indexer,
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
	deltafifo.Add(pod1)
	deltafifo.Add(pod2)
	deltafifo.Add(pod3)
	deltafifo.Pop(Process(indexer))
	deltafifo.Pop(Process(indexer))
	deltafifo.Pop(Process(indexer))

	// indexer.Add(pod1)
	// indexer.Add(pod2)
	// indexer.Add(pod3)
	pods, err := indexer.ByIndex("namespace", "kube-system")
	if err != nil {
		panic(err)
	}
	for _, s := range pods {
		fmt.Println(s.(*corev1.Pod).Name)
	}
	fmt.Println("----------")
	pods, err = indexer.ByIndex("nodename", "node1")
	if err != nil {
		panic(err)
	}
	for _, s := range pods {
		fmt.Println(s.(*corev1.Pod).Name)
	}
}
