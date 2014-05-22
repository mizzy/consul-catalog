package consulcatalog

import (
	"testing"
	"os"
)

func testClient(t *testing.T) *Client {
	client, err := NewClient(DefaultConfig())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return client
}

func TestService(t *testing.T) {
	client := testClient(t)
	meta, nodes, err := client.GetService("consul")

	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if meta.ModifyIndex == 0 {
		t.Fatalf("unexpected value: %#v", meta)
	}

	node        := nodes[0]
	hostname, _ := os.Hostname()
	if node.Node != hostname {
		t.Fatalf("unexpected return: %v", node.Node)
	}
	if node.ServiceName != "consul" {
		t.Fatalf("unexpected return: %v", node.ServiceName)
	}
}
