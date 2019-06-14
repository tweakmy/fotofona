package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/coreos/etcd/clientv3"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	CmdExecute()

	//Wait for process kill signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	config, err := clientcmd.BuildConfigFromFlags("", *flagKubeConfig)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	inf := NewInformer(*flagWatchLabels, clientset)
	go inf.Start(ctx) //Start Getting data right away

	//Create a new down stream lease
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"http://localhost:2378"},
		DialTimeout: 2 * time.Second,
	})

	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	lease := NewEtcdLease(cli)
	RunController(ctx, *flagEtcdRootPath, *flagKubeMasterDNSName, 10, lease, inf)
	// Block until a signal is received.
	<-c

}
