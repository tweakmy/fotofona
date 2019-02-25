package agent

import (
	"fmt"
	"context"
	"log"
	"github.com/coreos/etcd/clientv3"
	"time"
	"strconv"
)



// Updates Etcd Key Value for CoreDNS plugin
func StartAgent(){

    //Connecting to the cluster
    ctx, _ := context.WithTimeout(context.Background(), requestTimeout)
	cli, err := clientv3.New(clientv3.Config{
			DialTimeout: dialTimeout,
		Endpoints: []string{"127.0.0.1:2379"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

}

// Periodically updates the cluster with the dns IP name
func LeaseIP() {

    //nominated Domain Name by the user
    kubeMasterDomainName := os.Getenv("KUBE_MASTER_DOMAIN_NAME")
    //nominated DNS TTL for the client to refresh TTL eg 60 (seconds)
    kubeMasterDomainNameTTL := os.Getenv("KUBE_MASTER_DOMAIN_TTL")

    //This is used to detect which X{n} it should write it to
    masterName := os.Getenv("METADATANAME")
    masterHostIP := os.GetEnv("HOSTIP") //The assumption, the kubernetes is running using host ip

	kv.Delete(ctx, "key", clientv3.WithPrefix())

	gr, _ := kv.Get(ctx, "key")
	if len(gr.Kvs) == 0 {
		fmt.Println("No 'key'")
	}

	lease, err := cli.Grant(ctx, 1)
	if err != nil {
		log.Fatal(err)
	}

	// Insert key with a lease of 1 second TTL
	kv.Put(ctx, "key", "value", clientv3.WithLease(lease.ID))

	gr, _ = kv.Get(ctx, "key")
	if len(gr.Kvs) == 1 {
		fmt.Println("Found 'key'")
	}

	// Let the TTL expire
	time.Sleep(3 * time.Second)

	gr, _ = kv.Get(ctx, "key")
	if len(gr.Kvs) == 0 {
		fmt.Println("No more 'key'")
	}

}