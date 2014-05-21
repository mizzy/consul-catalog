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

func TestServices(t *testing.T) {
	client := testClient(t)
	meta, data, err := client.GetServices()

	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if meta.ModifyIndex == 0 {
		t.Fatalf("unexpected value: %#v", meta)
	}
	if data["consul"] != nil {
		t.Fatalf("unexpected return: %v", data)
	}
}

func TestService(t *testing.T) {
	client := testClient(t)
	meta, data, err := client.GetService("consul")

	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if meta.ModifyIndex == 0 {
		t.Fatalf("unexpected value: %#v", meta)
	}

	service     := data[0].(map[string]interface{})
	hostname, _ := os.Hostname()
	if service["Node"] != hostname {
		t.Fatalf("unexpected return: %v", service["Node"])
	}
	if service["ServiceName"] != "consul" {
		t.Fatalf("unexpected return: %v", service["ServiceName"])
	}
}
