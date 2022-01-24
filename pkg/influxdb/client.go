package influxdb

import (
	"github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type Client struct {
	influxCli influxdb2.Client
	writeAPI  api.WriteAPI
	queryAPI  api.QueryAPI
}

func NewClient() (*Client, error) {
	panic("TODO")
}

// Insert enclose logic for write data to influxDB
func (cli *Client) Insert() error {
	panic("TODO")
}

// QueryNodeGraph enclose logic for query node graph data from influxDB
func (cli *Client) QueryNodeGraph() (interface{}, error) {
	panic("TODO")
}

// QueryAnnotations enclose logic for query annotation data form influxDB
func (cli *Client) QueryAnnotations() (interface{}, error) {
	panic("TODO")
}
