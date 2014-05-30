package consulcatalog

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

type T struct {
	*testing.T
	server *httptest.Server
	client *Client
}

func testClient(t *testing.T) *Client {
	client, err := NewClient(DefaultConfig())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return client
}

func RunTestWithServer(t *testing.T, f func(xt *T)) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Static responses, ok?
			w.Header().Set("X-Consul-Index", "1")
			switch r.URL.Path {
			case "/v1/catalog/datacenters":
				w.Write([]byte(`["dc1", "dc2"]`))
			case "/v1/catalog/service/consul":
				w.Write([]byte(`[{
					"Node": "localhost",
	        "Address": "127.0.0.1",
	        "ServiceID": "consul",
	        "ServiceName": "consul",
	        "ServiceTags": null,
	        "ServicePort": 8000
	      }]`))
			default:
				w.WriteHeader(404)
			}
			return
		}),
	)
	defer server.Close()

	config := DefaultConfig()
	config.Address = server.URL[7:]
	client, err := NewClient(config)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	f(&T { t, server, client })
}

func TestService(tt *testing.T) {
	RunTestWithServer(tt, func(t *T) {
		client := t.client
		nodes, err := client.GetService("consul")

		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if meta := nodes.Meta(); meta.ModifyIndex == 0 {
			t.Fatalf("unexpected value: %#v", meta)
		}

		node := nodes.NodeAt(0)
		if node.Node != "localhost" {
			t.Fatalf("unexpected return: %v", node.Node)
		}
		if node.ServiceName != "consul" {
			t.Fatalf("unexpected return: %v", node.ServiceName)
		}
	})
}

func TestNonExistentService(tt *testing.T) {
	RunTestWithServer(tt, func(t *T) {
		client := t.client
		nodes, err := client.GetService("foobar")

		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if meta := nodes.Meta(); meta.ModifyIndex == 0 {
			t.Fatalf("unexpected value: %#v", meta)
		}

		if nodes.IsValid() {
			t.Fatalf("should be invalid")
		}
	})
}

func TestDatacenters(tt *testing.T) {
	RunTestWithServer(tt, func(t *T) {
		client := t.client

		dcs, err := client.GetDatacenters()

		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if meta := dcs.Meta(); meta.ModifyIndex == 0 {
			t.Fatalf("unexpected value: %#v", meta)
		}

		if !reflect.DeepEqual(dcs.Names(), []Datacenter { "dc1", "dc2" }) {
			t.Fatalf("unexpected value: %#v", dcs)
		}
	})
}
