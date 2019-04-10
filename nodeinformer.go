package main

import (
	"context"
	"fmt"
	"sync"

	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
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

	factory := informers.NewFilteredSharedInformerFactory(clientset, 0, "", func(o *metaV1.ListOptions) {
		o.LabelSelector = "node-role.kubernetes.io/master="
	})

	nodeInformer := factory.Core().V1().Nodes().Informer()

	i.lister = factory.Core().V1().Nodes().Lister()

	nodeInformer.AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				i.updateIPs()
			},
			UpdateFunc: func(oldobj, newObj interface{}) {
				i.updateIPs()
			},
			DeleteFunc: func(obj interface{}) {
				i.updateIPs()
			},
		})

	factory.Start(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), nodeInformer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Timed out waiting for caches to sync"))
		return
	}

	fmt.Println("cache is synced")

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
