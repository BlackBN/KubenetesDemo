package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"time"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Controller struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		indexer:  indexer,
		queue:    queue,
		informer: informer,
	}
}

func (c *Controller) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	//关闭队列
	defer c.queue.ShutDown()

	klog.Infof("starting controller")
	// 启动 informer
	go c.informer.Run(stopCh)

	// 需要等待缓存同步
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
	}
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}
	<-stopCh
}

func (c *Controller) runWorker() {
	//
	for c.processNextItem() {

	}
}

func (c *Controller) processNextItem() bool {
	// 从队列里面取出一个元素
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// 告诉队列已经处理了该元素
	defer c.queue.Done(key)
	err := c.logic(key.(string))
	c.handleErr(err, key)
	return true
}

func (c *Controller) logic(key string) error {
	obj, exist, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("get obj from indexer key %v is error %v", key, err)
		return err
	}
	if !exist {
		fmt.Printf("obj %v is not exist\n", key)
	} else {
		fmt.Printf("obj %v for pod", obj.(*v1.Deployment))
	}

	return nil
}

func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		c.queue.Forget(key)
		return
	}
	if c.queue.NumRequeues(key) < 5 {
		c.queue.AddRateLimited(key)
		return
	}
	c.queue.Forget(key)
	runtime.HandleError(err)
}

func initClient() (*kubernetes.Clientset, error) {
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
			return nil, err
		}
	}
	return kubernetes.NewForConfig(config)
}

func main() {
	clientSet, err := initClient()
	if err != nil {
		klog.Fatal(err)
	}
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	indexer, informer := cache.NewIndexerInformer(
		cache.NewListWatchFromClient(
			clientSet.AppsV1().RESTClient(),
			"deployments",
			corev1.NamespaceAll,
			fields.Everything(),
		),
		&v1.Deployment{},
		0*time.Second,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Println("add func")
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					queue.Add(key)
				}
			},
			DeleteFunc: func(obj interface{}) {
				fmt.Println("delete func")
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					queue.Add(key)
				}

			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				fmt.Println("update func")
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				if err == nil {
					queue.Add(key)
				}
			},
		}, cache.Indexers{})

	controller := NewController(queue, indexer, informer)
	stopCh := make(chan struct{})
	defer close(stopCh)
	go controller.Run(1, stopCh)
	select {}
}
