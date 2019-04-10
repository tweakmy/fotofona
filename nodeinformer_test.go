package main

import (
	"context"
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

func TestInformer(t *testing.T) {

	var nodes = v1.Node{
		ObjectMeta: metaV1.ObjectMeta{
			Labels: map[string]string{
				"node-role.kubernetes.io/master": "",
			},
		},
		Status: v1.NodeStatus{
			Addresses: []v1.NodeAddress{
				v1.NodeAddress{
					Type:    v1.NodeAddressType("InternalIP"),
					Address: "10.0.0.1",
				},
			},
		},
	}

	ctx, cancel := context.WithCancel(context.TODO())

	inf := Informer{}

	// Create the fake client.
	client := fake.NewSimpleClientset(&nodes)

	inf.StartInformer(ctx, client, false)

	time.Sleep(3 * time.Second)

	cancel()
}
