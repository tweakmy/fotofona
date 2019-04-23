package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// NodeAddress struct {
// 	// Node address type, one of Hostname, ExternalIP or InternalIP.
// 	Type NodeAddressType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=NodeAddressType"`
// 	// The node address.
// 	Address

func NewMasterNode(nodeName string, ipaddress string) v1.Node {
	return v1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				v1.NodeAddress{
					Type:    v1.NodeAddressType("InternalIP"),
					Address: ipaddress,
				},
			},
		},
	}
}

func TestInformer(t *testing.T) {

	var nodes = NewMasterNode("node1", "10.0.0.1")

	ctx, cancel := context.WithCancel(context.TODO())

	inf := Informer{}

	// Create the fake client.
	client := fake.NewSimpleClientset(&nodes)

	go inf.StartInformer(ctx, client, false)

	time.Sleep(3 * time.Second)

	var node2 = NewMasterNode("node2", "10.0.0.2")

	client.CoreV1().Nodes().Create(&node2)

	//fmt.Println(err)
	time.Sleep(3 * time.Second)
	ips, _ := inf.GetHostIPs()

	fmt.Println(ips)

	cancel()
}
