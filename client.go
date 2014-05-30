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
	Datacenter Datacenter

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

// CatalogResponse is the interface enveloping the whole set of response types
// The only common operation is Meta() and IsValid() methods
type CatalogResponse interface {
	// IsValid tells if the response was an empty one (where 404 was returned)
	// so you have a valid CatalogMeta, but no real result
	IsValid() bool

	// Meta returns the CatalogMeta object
	Meta() *CatalogMeta

	makeInvalid()
}

type CatalogMeta struct {
	ModifyIndex uint64
}

func (c *CatalogMeta) Meta() *CatalogMeta {
	return c
}

func (c *CatalogMeta) Parse(resp *http.Response) (error) {
	// Decode the CatalogMeta
	index, err := strconv.ParseUint(resp.Header.Get("X-Consul-Index"), 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse X-Consul-Index: %v", err)
	}

	c.ModifyIndex = index
	return nil
}

type Datacenter string
type Datacenters struct {
	validResponse
	*CatalogMeta
	centers []Datacenter
}

func (d Datacenter) String() string {
		return string(d)
}

func (d *Datacenters) Names() []Datacenter {
	return d.centers
}

func (d *Datacenters) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &d.centers); err != nil {
		return err
	}
	return nil
}

type Node struct {
	Node        string
	Address     string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	ServicePort int
}

type validResponse bool

func (v *validResponse) makeInvalid() {
	*v = false
}

func (v validResponse) IsValid() bool {
	return bool(v)
}

type Nodes struct {
	validResponse
	*CatalogMeta
	nodes []*Node
}

func (n *Nodes) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &n.nodes); err != nil {
		return err
	}
	return nil
}

func (n *Nodes) NodeAt(i int) *Node {
	return n.nodes[i]
}

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
func (c *Client) GetService(service string) (*Nodes, error) {
	r := &Nodes{ true, &CatalogMeta{}, []*Node{} }
	err := c.request(
		c.pathURL(0, "service", service),
		r,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) GetDatacenters() (*Datacenters, error) {
	r := &Datacenters{ true, &CatalogMeta{}, []Datacenter{} }
	err := c.request(
		c.pathURL(0, "datacenters"),
		r,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (c *Client) request(url *url.URL, r CatalogResponse) error { 
	req := &http.Request{
		Method: "GET",
		URL:    url,
	}

	resp, err := c.config.HTTPClient.Do(req)
	if err != nil {
		return err
	}

	if err := r.Meta().Parse(resp); err != nil {
		return err
	}

	// Ensure status code is 404 or 200
	if resp.StatusCode == 404 {
		r.makeInvalid()
		return nil
	}

  if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected response code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	if err = dec.Decode(r); err != nil {
		return err
	}

	return nil
}

// path is used to generate the HTTP path for a request
func (c *Client) pathURL(waitIndex uint64, paths ...string) *url.URL {
	url := &url.URL{
		Scheme: "http",
		Host:   c.config.Address,
		Path:   "/v1/catalog/" + path.Join(paths...),
	}
	query := url.Query()

	if c.config.Datacenter != "" {
		query.Set("dc", c.config.Datacenter.String())
		url.RawQuery = query.Encode()
	}

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
	return url
}

