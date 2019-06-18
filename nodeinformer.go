package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/golang/glog"

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

	//lister
	lister v1.NodeLister

	//workqueue
	queue workqueue.RateLimitingInterface

	indexer cache.Indexer

	// Error counter
	errorCount int

	//Closing Down Channel due to comms error
	errCloseChan chan struct{}

	//WatchLabels - comma separated label a=x,b=y
	watchLabels string
}

// NewInformer - Create a new Informer
func NewInformer(watchLabels string, clientset kubernetes.Interface) *Informer {
	return &Informer{
		//Initialize the channel
		updateHostIPsChan: make(chan struct{}),
		queue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		errCloseChan:      make(chan struct{}),
		watchLabels:       watchLabels,
		clientset:         clientset,
	}
}

// GetHostIPs - List all the IPs
func (i *Informer) GetHostIPs(ctx context.Context) (hostips []string, err error) {
	glog.Infoln("Read host ips")
	defer i.rwLock.Unlock()
	i.rwLock.Lock()
	return i.hostsIPs, nil
}

// GetNodeAddress - Return the node
func GetNodeAddress(node *v1Api.Node, matchAddressType string) (ipaddress string, ConditionReady bool, err error) {

	for _, conditions := range node.Status.Conditions {

		if string(conditions.Type) == "Ready" {
			if string(conditions.Status) == "True" {
				//fmt.Println(address.Address)
				ConditionReady = true
			} else {
				ConditionReady = false
			}
		}

	}

	for _, address := range node.Status.Addresses {

		if string(address.Type) == matchAddressType {

			ipaddress = string(address.Address)
			return
		}
	}

	return "", false, fmt.Errorf("Cannot locate IP Address Type: %s", matchAddressType)
}

// updateHostIPS - update the IPs to local var
func (i *Informer) updateHostIPs() error {

	nodes, err := i.lister.List(labels.Everything())

	if err != nil {
		return err
	}

	i.hostsIPs = []string{}

	for _, node := range nodes {

		nodeip, nodeisready, err := GetNodeAddress(node, addressType)
		if err != nil {
			panic(err)
		}

		if nodeisready {
			i.hostsIPs = append(i.hostsIPs, nodeip)
		}

	}

	sort.Strings(i.hostsIPs) //Make sure IP is in ascending mode

	return nil
}

// GetInformerInterupt - Provide the downstream api a notify that there was a change
func (i *Informer) GetInformerInterupt() chan struct{} {
	return i.updateHostIPsChan
}

// GetInformerErrorClose - Provide upstream that the informer can no longer proceed
func (i *Informer) GetInformerErrorClose() chan struct{} {
	return i.errCloseChan
}

// Start - connect the kubernetes master
func (i *Informer) Start(ctx context.Context) {

	ctx, cancel := context.WithCancel(ctx)

	//Added error count
	errorCountFunc := func(err error) {

		if strings.Contains(err.Error(), "connect: connection refused") || strings.Contains(err.Error(), "error") ||
			strings.Contains(err.Error(), "Failed") {
			i.errorCount++
			glog.Infof("Error count %d\n", i.errorCount)
		}

		if i.errorCount > 2 {
			cancel()
			i.errCloseChan <- struct{}{} //Trigger up chain, that comms issues
			glog.Errorf("Terminiating due to error")

		}
	}

	//Add additional Error Handling
	//if len(runtime.ErrorHandlers) > 2 {
	runtime.ErrorHandlers = runtime.ErrorHandlers[0:2]
	runtime.ErrorHandlers = append(runtime.ErrorHandlers, errorCountFunc)
	//}

	if i.clientset == nil {
		panic("Client is not properly setup")
	}

	i.rwLock.Lock() //Block the downstream from reading intially
	//fmt.Println("Lock write")

	factory := informers.NewFilteredSharedInformerFactory(i.clientset, 0, "", func(o *metaV1.ListOptions) {
		o.LabelSelector = i.watchLabels
	})

	nodeInformer := factory.Core().V1().Nodes().Informer()

	i.lister = factory.Core().V1().Nodes().Lister()
	i.indexer = factory.Core().V1().Nodes().Informer().GetIndexer()

	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {

				key, err := cache.MetaNamespaceKeyFunc(obj)
				if err == nil && nodeInformer.HasSynced() {
					glog.V(2).Infof("Node added %s", key)
					i.queue.Add(key)
				}
			},
			UpdateFunc: func(oldobj, newObj interface{}) {

				key, err := cache.MetaNamespaceKeyFunc(newObj)
				//key2, _ := cache.DeletionHandlingMetaNamespaceKeyFunc(oldobj)
				if err == nil && nodeInformer.HasSynced() {
					glog.V(2).Infof("Node updated %s", key)
					i.queue.Add(key)
				}

			},
			DeleteFunc: func(obj interface{}) {
				key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
				if err == nil && nodeInformer.HasSynced() {
					glog.V(2).Infof("Node deleted %s", key)
					i.queue.Add(key)
				}
			},
		})

	factory.Start(ctx.Done())

	//fmt.Println("before cache is synced", i.hostsIPs)

	if !cache.WaitForCacheSync(ctx.Done(), nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		i.errCloseChan <- struct{}{}
		return
	}

	//Attemp to do the initial update
	err := i.updateHostIPs()
	if err != nil {
		runtime.HandleError(fmt.Errorf("Lister could not be read"))
		return
	}
	i.rwLock.Unlock() //Now the downstream can read the first listing
	//fmt.Println("Unlock write")

	glog.Infof("cache is synced %s", i.hostsIPs)

	threadiness := 1

	for j := 0; j < threadiness; j++ {
		go wait.Until(i.runWorker, time.Second, ctx.Done())
	}

	<-ctx.Done()
	glog.Infof("Stop Informer")

}

func (i *Informer) runWorker() {
	for i.processNextItem() {
	}
}

func (i *Informer) processNextItem() bool {

	glog.V(2).Infof("Process next item")

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
		glog.Infof("%s Fetching object with key %s from store failed with %v\n", time.Now().String(), key, err)
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

		glog.V(2).Infof("Got hostips %q", i.hostsIPs)

		i.updateHostIPsChan <- struct{}{} //Notify downstream to start reacting

	} else {
		//Could be Add/Sync/Update
		hostsIPsStr := fmt.Sprintf("%q", i.hostsIPs)

		node := obj.(*v1Api.Node)
		nodeIP, nodeisready, err2 := GetNodeAddress(node, addressType)
		if err != nil {
			panic(err2)
		}

		existInHostsIPsList := strings.Contains(hostsIPsStr, nodeIP)

		if (nodeisready && !existInHostsIPsList) || (!nodeisready && existInHostsIPsList) {

			i.rwLock.Lock()
			err := i.updateHostIPs()
			i.rwLock.Unlock()
			if err != nil {
				runtime.HandleError(fmt.Errorf("Lister could not be read"))
				return false
			}

			glog.V(2).Infof("Got hostips %q", i.hostsIPs)
			i.updateHostIPsChan <- struct{}{} //Notify downstream to start reacting
		}

	}

	return true
}
