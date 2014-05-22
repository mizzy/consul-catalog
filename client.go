package consulcatalog

import (
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
	"encoding/json"
)

// Config is used to configure the creation of a client
type Config struct {
	// Address is the address of the Consul server
	Address string

	// Datacenter to use. If not provided, the default agent datacenter is used.
	Datacenter string

	// HTTPClient is the client to use. Default will be
	// used if not provided.
	HTTPClient *http.Client

	// WaitTime limits how long a Watch will block. If not provided,
	// the agent default values will be used.
	WaitTime time.Duration
}

// Client provides a client to Consul for K/V data
type Client struct {
	config Config
}

// CatalogMeta provides meta data about a query
type CatalogMeta struct {
	ModifyIndex uint64
}

type Node struct {
	Node        string
	Address     string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	ServicePort int
}

type Nodes []*Node

// NewClient returns a new
func NewClient(config *Config) (*Client, error) {
	client := &Client{
		config: *config,
	}
	return client, nil
}

// DefaultConfig returns a default configuration for the client
func DefaultConfig() *Config {
	return &Config{
		Address:    "127.0.0.1:8500",
		HTTPClient: http.DefaultClient,
	}
}

// Get nodes that have a service
func (c *Client) GetService(service string) (*CatalogMeta, Nodes, error) {
	meta, nodes, err := c.Get("service", service, 0)
	return meta, nodes, err
}

// GET
func (c *Client) Get(endpoint string, path string, waitIndex uint64) (*CatalogMeta, Nodes, error) {
	url := c.pathURL(endpoint, path)
	query := url.Query()

	if waitIndex > 0 {
		query.Set("index", strconv.FormatUint(waitIndex, 10))
	}
	if waitIndex > 0 && c.config.WaitTime > 0 {
		waitMsec := fmt.Sprintf("%dms", c.config.WaitTime/time.Millisecond)
		query.Set("wait", waitMsec)
	}
	if len(query) > 0 {
		url.RawQuery = query.Encode()
	}
	req := http.Request{
		Method: "GET",
		URL:    url,
	}
	resp, err := c.config.HTTPClient.Do(&req)
	if err != nil {
		return nil, nil, err
	}

	// Decode the CatalogMeta
	meta := &CatalogMeta{}
	index, err := strconv.ParseUint(resp.Header.Get("X-Consul-Index"), 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse X-Consul-Index: %v", err)
	}
	meta.ModifyIndex = index

	// Ensure status code is 404 or 200
	if resp.StatusCode == 404 {
		return meta, nil, nil
	} else if resp.StatusCode != 200 {
		return nil, nil, fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	var out Nodes
	if err := dec.Decode(&out); err != nil {
		return nil, nil, err
	}

	return meta, out, nil
}

// path is used to generate the HTTP path for a request
func (c *Client) pathURL(path0 string, path1 string) *url.URL {
	url := &url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   path.Join("/v1/catalog/", path0, path1),
	}
	if c.config.Datacenter != "" {
		query := url.Query()
		query.Set("dc", c.config.Datacenter)
		url.RawQuery = query.Encode()
	}
	return url
}

