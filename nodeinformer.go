package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"

	v1Api "k8s.io/api/core/v1"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
)

const addressType = "InternalIP"

// Informer - Kubernetes Operator to
type Informer struct {

	//kubernetes client settings
	clientset kubernetes.Interface

	//Signals the downstream api to update hostips
	updateHostIPsChan chan struct{}

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

// NewInformer - Create a new Informer
func NewInformer() *Informer {
	return &Informer{
		//Initialize the channel
		updateHostIPsChan: make(chan struct{}),
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
	}
}

// GetHostIPs - List all the IPs
func (i *Informer) GetHostIPs() (hostips []string) {
	fmt.Println("Read host ips")
	defer i.rwLock.Unlock()
	i.rwLock.Lock()
	return i.hostsIPs
}

// updateHostIPS - update the IPs to local var
func (i *Informer) updateHostIPs() error {

	nodes, err := i.lister.List(labels.Everything())

	if err != nil {
		return err
	}

	i.hostsIPs = []string{}

	for _, node := range nodes {
		nodeip, err := GetNodeAddress(node, addressType)
		if err != nil {
			panic(err)
		}

		i.hostsIPs = append(i.hostsIPs, nodeip)

	}

	return nil
}

// GetInformerInterupt - Provide the downstream api a notify that there was a change
func (i *Informer) GetInformerInterupt() chan struct{} {
	return i.updateHostIPsChan
}

// SetupClient - Split the function between setting up the client and controller
func (i *Informer) SetupClient(useKubeConfig bool) {

	kubeconfig := filepath.Join(
		os.Getenv("HOME"), ".kube", "config",
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	i.clientset = clientset
}

// Start - connect the kubernetes master
func (i *Informer) Start(ctx context.Context) {

	if i.clientset == nil {
		panic("Client is not properly setup")
	}

	i.rwLock.Lock() //Block the downstream from reading intially
	fmt.Println("Lock write")

	factory := informers.NewFilteredSharedInformerFactory(i.clientset, 0, "", func(o *metaV1.ListOptions) {
		o.LabelSelector = "node-role.kubernetes.io/master="
	})

	nodeInformer := factory.Core().V1().Nodes().Informer()

	i.lister = factory.Core().V1().Nodes().Lister()
	i.indexer = factory.Core().V1().Nodes().Informer().GetIndexer()

	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil && nodeInformer.HasSynced() {
					fmt.Println("Add", key)
					i.queue.Add(key)
				}
			},
			UpdateFunc: func(oldobj, newObj interface{}) {

				key, err := cache.MetaNamespaceKeyFunc(newObj)
				key2, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(oldobj)
				if err == nil && nodeInformer.HasSynced() {
					fmt.Println("Updated", key2, key)
					i.queue.Add(key)
				}

			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil && nodeInformer.HasSynced() {
					fmt.Println("Deleted", key)
					i.queue.Add(key)
				}
			},
		})

	factory.Start(ctx.Done())

	fmt.Println("before cache is synced", i.hostsIPs)

	if !cache.WaitForCacheSync(ctx.Done(), nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	//Attemp to do the initial update
	err := i.updateHostIPs()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Lister could not be read"))
		return
	}
	i.rwLock.Unlock() //Now the downstream can read the first listing
	fmt.Println("Unlock write")

	fmt.Println("cache is synced", i.hostsIPs)

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
	fmt.Println("Process next item")

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
		//There is a deletion that just happened
		i.rwLock.Lock()
		err := i.updateHostIPs()
		i.rwLock.Unlock()
		if err != nil {
			runtime.HandleError(fmt.Errorf("Lister could not be read"))
			return false
		}
		fmt.Printf("%q\n", i.hostsIPs)
		i.updateHostIPsChan <- struct{}{} //Notify downstream to start reacting

	} else {
		//Could be Add/Sync/Update
		hostsIPsStr := fmt.Sprintf("%q", i.hostsIPs)

		node := obj.(*v1Api.Node)
		nodeIP, err2 := GetNodeAddress(node, addressType)
		if err != nil {
			panic(err2)
		}

		if !strings.Contains(hostsIPsStr, nodeIP) {

			i.rwLock.Lock()
			err := i.updateHostIPs()
			i.rwLock.Unlock()
			if err != nil {
				runtime.HandleError(fmt.Errorf("Lister could not be read"))
				return false
			}
			fmt.Printf("%q\n", i.hostsIPs)
			i.updateHostIPsChan <- struct{}{} //Notify downstream to start reacting
		}
	}

	return true
}

// GetNodeAddress - Return the node
func GetNodeAddress(node *v1Api.Node, matchAddressType string) (string, error) {

	for _, address := range node.Status.Addresses {

		if string(address.Type) == matchAddressType && node.Status.Phase == "Ready" {
			//fmt.Println(address.Address)
			return string(address.Address), nil
		}
	}

	return "", fmt.Errorf("Cannot locate IP Address Type: %s", matchAddressType)
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
