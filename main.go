package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/coreos/etcd/clientv3"
	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {

	RootCmd.Run = func(cmd *cobra.Command, args []string) {

		//Sets default for the glog for now
		flag.Set("alsologtostderr", fmt.Sprintf("%t", true))
		flag.Set("v", "2")

		//Validate all the user input
		if *flagEtcdRootPath == "" {
			glog.Errorf("--rootpath: must not be empty")
			os.Exit(0)
		}

		if govalidator.IsDNSName(*flagKubeMasterDomainName) {
			glog.Errorf("--domainname: should use qualified domain name")
			os.Exit(0)
		}

		kubeconfig := *flagKubeConfig
		if *flagUseKubeConfig {
			if _, err := os.Stat(*flagKubeConfig); os.IsNotExist(err) {
				glog.Errorf("--kubeconfigpath: kubeconfig path must exist")
				os.Exit(0)
			}

		} else {
			kubeconfig = "" //Forces it use service account
		}

		//Wait for process kill signal
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, os.Kill)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			glog.Fatal(err)
			os.Exit(1)
		}

		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			glog.Fatal(err)
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
			glog.Fatal(err)
			os.Exit(1)
		}

		lease := NewEtcdLease(cli)
		RunController(ctx, *flagEtcdRootPath, *flagKubeMasterDomainName, 10, lease, inf)
		// Block until a signal is received.
		<-c

	}

	CmdExecute()

}
