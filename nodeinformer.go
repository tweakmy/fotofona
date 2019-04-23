package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	apiV1 "k8s.io/api/core/v1"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
)

// Informer - Kubernetes Operator to
type Informer struct {

	//If this was valid
	validIPs bool

	//List of the master host ips to be written
	hostsIPs []string

	//RW Lock in case, external issues a read
	rwLock sync.RWMutex

	// lister
	lister v1.NodeLister

	//workqueue
	queue workqueue.RateLimitingInterface

	indexer cache.Indexer
}

// GetHostIPs - List all the IPs
func (i *Informer) GetHostIPs() (hostips []string, err error) {

	nodelist, _ := i.lister.List(labels.Everything())

	for _, node := range nodelist {
		//fmt.Println(node.Name)
		for _, address := range node.Status.Addresses {
			if string(address.Type) == "InternalIP" {
				//fmt.Println(address.Address)
				hostips = append(hostips, string(address.Address))

			}
		}
	}

	return
}

// updateIPs - run func to update the IPs on the informer
func (i *Informer) updateIPs() {

	//Release for external read
	i.rwLock.RLock()

	//Blank it out
	i.hostsIPs = []string{}

	// "k8s.io/apimachinery/pkg/apis/meta/v1" provides an Object
	// interface that allows us to get metadata easily
	nodelist, _ := i.lister.List(labels.Everything())

	for _, node := range nodelist {
		//fmt.Println(node.Name)
		for _, address := range node.Status.Addresses {
			if string(address.Type) == "InternalIP" {
				//fmt.Println(address.Address)
				i.hostsIPs = append(i.hostsIPs, string(address.Address))
				fmt.Println("Update ip host", i.hostsIPs)
			}
		}
	}

	//Release for external read
	i.rwLock.RUnlock()
}

// StartInformer - connect the kubernetes master
func (i *Informer) StartInformer(ctx context.Context, clientset kubernetes.Interface, useKubeConfig bool) {

	i.queue = workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	factory := informers.NewFilteredSharedInformerFactory(clientset, 0, "", func(o *metaV1.ListOptions) {
		o.LabelSelector = "node-role.kubernetes.io/master="
	})

	nodeInformer := factory.Core().V1().Nodes().Informer()

	i.lister = factory.Core().V1().Nodes().Lister()
	i.indexer = factory.Core().V1().Nodes().Informer().GetIndexer()

	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				fmt.Println("Add")
				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil {
					i.queue.Add(key)
				}
			},
			UpdateFunc: func(oldobj, newObj interface{}) {
				fmt.Println("Updated")
				key, err := cache.MetaNamespaceKeyFunc(newObj)
				if err == nil {
					i.queue.Add(key)
				}

			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil {
					i.queue.Add(key)
				}
			},
		})

	factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	fmt.Println("cache is synced")

	threadiness := 1

	for j := 0; j < threadiness; j++ {
		go wait.Until(i.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	fmt.Println("Stop Informer")

}

func (i *Informer) runWorker() {
	for i.processNextItem() {
	}
}

func (i *Informer) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := i.queue.Get()
	if quit {
		return false
	}
	// Tell the queue that we are done with processing this key. This unblocks the key for other workers
	// This allows safe parallel processing because two pods with the same key are never processed in
	// parallel.
	defer i.queue.Done(key)

	obj, exists, err := i.indexer.GetByKey(key.(string))

	if err != nil {
		fmt.Printf("Fetching object with key %s from store failed with %v", key, err)

	}

	if !exists {
		// Below we will warm up our cache with a Pod, so that we will see a delete for one pod
		fmt.Printf("Pod %s does not exist anymore\n", key)
	} else {
		// Note that you also have to check the uid if you have a local controlled resource, which
		// is dependent on the actual instance, to detect that a Pod was recreated with the same name
		//fmt.Printf("Sync/Add/Update for Pod %s\n", obj.(*Node).GetName())
		fmt.Println(obj.(*apiV1.Node).GetName(), obj.(*apiV1.Node).GetResourceVersion())
	}

	return true
}

// // StartInformerDoNOTUSE - do not use
// func (i *Informer) StartInformerDoNOTUSE(ctx context.Context, useKubeConfig bool) {

// 	kubeconfig := filepath.Join(
// 		os.Getenv("HOME"), ".kube", "config",
// 	)

// 	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	clientset, err := kubernetes.NewForConfig(config)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	//Only interested in the master node
// 	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{
// 		LabelSelector: "node-role.kubernetes.io/master=",
// 	})

// 	if err != nil {
// 		log.Fatalln("failed to get nodes:", err)
// 	}

// 	for i, node := range nodes.Items {
// 		fmt.Printf("[%d] %s\n", i, node.GetName())
// 		for _, address := range node.Status.Addresses {
// 			if string(address.Type) == "InternalIP" {
// 				fmt.Println(address.Address)
// 			}
// 		}
// 	}
// }
