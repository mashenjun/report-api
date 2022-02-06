package main

import (
	"context"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	http2 "github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/pingcap/log"
	"go.uber.org/zap"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"time"
)

type ReportAPIOption func(reportAPI *ReportAPI) error

type ReportAPI struct {
	bucket string
	org    string

	influxCli  influxdb2.Client
	writeAPI   api.WriteAPI
	queryAPI   api.QueryAPI
	writeErrCh <-chan error

	// internal variable
	done chan struct{}
}

func NewReportAPI(endpoint string, org string, bucket string, token string, opts ...ReportAPIOption) (*ReportAPI, error) {
	//influxdbURL := "http://localhost:8086"
	//tk := "lF9VJ9pvM3xU4piExllnV800kHwGg9ie-08fTnlcZl9EcYllFMo9urMGTQ71AI3UKTJTn5D6LiRmCvDLAG9BPQ=="
	//org := "my-org"
	//bucket := "clinic"

	rAPI := &ReportAPI{
		bucket: bucket,
		org:    org,
		done:   make(chan struct{}),
	}

	for _, opt := range opts {
		if err := opt(rAPI); err != nil {
			return nil, err
		}
	}

	rAPI.influxCli = influxdb2.NewClient(endpoint, token)
	rAPI.writeAPI = rAPI.influxCli.WriteAPI(org, bucket)
	rAPI.writeAPI.SetWriteFailedCallback(retryCallBack)
	rAPI.writeErrCh = rAPI.writeAPI.Errors()
	rAPI.queryAPI = rAPI.influxCli.QueryAPI(org)

	go rAPI.writeErrorLoop()

	return rAPI, nil
}

// Close must be called by the caller
func (api *ReportAPI) Close() {
	if api == nil {
		return
	}
	api.writeAPI.Flush()
	api.influxCli.Close()
	close(api.done)
}

func (api *ReportAPI) QueryNodeGraph(ctx context.Context, param *QueryNodeGraphParam) (*QueryNodeGraphData, error) {
	// build the flux query
	fluxQueryBase := `
from(bucket: "%s")
	|> range(start: %v, stop: %v)
	|> filter(fn:(r) => r._measurement =="fast-tune-similarity" and r.tidb_cluster_id == "%v")
	|> group(columns: ["id"])
	|> first()
	|> filter(fn:(r) => r._value >= 0.5)
	|> sort(columns: ["id"])
`
	fluxQuery := fmt.Sprintf(fluxQueryBase, api.bucket, param.StartTS, param.EndTS, param.TiDBClusterID)
	fmt.Println(fluxQuery)
	result, err := api.queryAPI.Query(ctx, fluxQuery)
	if err != nil {
		log.Error("query influxdb failed", zap.Error(err))
		return nil, err
	}
	defer result.Close()

	data := QueryNodeGraphData{
		Nodes: make([]*Node, 0),
		Edges: make([]*Edge, 0),
	}

	nodesLookup := make(map[int64]struct{})
	for result.Next() {
		rd := result.Record()
		similarity, ok := rd.Value().(float64)
		if !ok {
			continue
		}

		idStr, ok := rd.ValueByKey("id").(string)
		if !ok {
			continue
		}
		id, err := strconv.ParseInt(idStr, 0, 64)
		if err != nil {
			log.Error("parse int failed", zap.Error(err))
			return nil, err
		}
		title, ok := rd.ValueByKey("title").(string)
		if !ok {
			title = "unknown"
		}

		node := DefaultNode()
		node.ID = idStr
		node.Title = idStr
		node.SubTitle = title
		node.MainStat = fmt.Sprintf("%.3f", similarity)
		node.ArcPositive = similarity
		node.ArcNegative = 1 - similarity

		data.Nodes = append(data.Nodes, node)
		nodesLookup[id] = struct{}{}
	}
	log.Info("", zap.Any("nodesLookuo", nodesLookup))
	for _, node := range data.Nodes {
		id, _ := strconv.ParseInt(node.ID, 0, 64)
		if targets, ok := EdgeMatrixV2[id]; ok {
			log.Info("", zap.Any("targets", targets), zap.Any("id", id))
			for _, target := range targets {
				if _, ok := nodesLookup[target]; !ok {
					continue
				}
				edge := DefaultEdge()
				// edge.ID = fmt.Sprintf("%#x", id<<16+target)
				edge.ID = fmt.Sprintf("%v%v", id, target)
				edge.Source = node.ID
				edge.Target = fmt.Sprintf("%v", target)
				data.Edges = append(data.Edges, edge)
			}
		}
	}

	if result.Err() != nil {
		log.Error("query parsing failed", zap.Error(result.Err()))
		return nil, err
	}
	return &data, nil
}

func (api *ReportAPI) QueryAnnotations(ctx context.Context, param *QueryAnnotationsParam) (QueryAnnotationsData, error) {
	fluxQueryBase := `
from(bucket: "%s")
	|> range(start: %v, stop: %v)
	|> filter(fn:(r) => r._measurement =="fast-tune-anomaly" and r.tidb_cluster_id == "%v")
`
	fluxQuery := fmt.Sprintf(fluxQueryBase, api.bucket, param.StartTS, param.EndTS, param.TiDBClusterID)

	result, err := api.queryAPI.Query(ctx, fluxQuery)
	if err != nil {
		log.Error("query influxdb failed", zap.Error(err))
		return nil, err
	}
	defer result.Close()

	data := make(QueryAnnotationsData, 0)
	for result.Next() {
		item := QueryAnnotationItem{
			Annotation: DefaultAnomalyAnnotation(),
			Time:       0,
			TimeEnd:    0,
			Title:      "",
			Tags:       "",
			Text:       "",
		}
		rd := result.Record()
		// Time should be milliseconds
		item.Time = rd.Time().UnixNano() / 1e6
		log.Info("", zap.Any("values", rd.Values()))
		if rd.Field() == "end_time" {
			if endTs, ok := rd.Value().(float64); ok {
				item.TimeEnd = int64(endTs) * 1e3
			}
		}
		item.Title = "anomaly title"
		item.Tags = "anomaly tags"
		data = append(data, item)
	}

	if result.Err() != nil {
		log.Error("query parsing failed", zap.Error(result.Err()))
		return nil, err
	}
	return data, nil
}

// TODO(shenjun): how to handle the error with async write?
func (api *ReportAPI) InsertSample(ctx context.Context, param *InsertSampleParam) (*InsertSampleData, error) {
	log.Info("InsertSample", zap.Any("param", param))
	ts := time.Unix(param.Timestamp, 0)
	point := influxdb2.NewPoint(param.Measurement, param.GetTags(), param.Fields, ts)
	api.writeAPI.WritePoint(point)
	return &InsertSampleData{}, nil
}

func (api *ReportAPI) Flush(ctx context.Context) error {
	api.writeAPI.Flush()
	return nil
}

// writeErrorLoop drain all write error for async write
func (api *ReportAPI) writeErrorLoop() {
	for {
		select {
		case <-api.done:
			return
		case err := <-api.writeErrCh:
			log.Error("write point failed", zap.Error(err))
		}
	}
}

func retryCallBack(batch string, err http2.Error, retryAttempts uint) bool {
	// if retry attempts more than 3, log and skip this retry
	if retryAttempts < 3 {
		return true
	}
	log.Error("send batch to influxdb failed", zap.Error(err.Err))
	return false
}

type DataAPI struct {
	endpoint string
	// host does not contain scheme
	host         string
	reserveProxy *httputil.ReverseProxy
}

func NewDataAPI(endpoint string) (*DataAPI, error) {
	u, err := url.ParseRequestURI(endpoint)
	if err != nil {
		return nil, err
	}

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = u.Host
		req.URL.Path = "/api/v1/query_range"
	}

	dAPI := &DataAPI{
		endpoint:     endpoint,
		host:         u.Host,
		reserveProxy: &httputil.ReverseProxy{Director: director},
	}

	return dAPI, nil
}

func (api *DataAPI) GetMetricsFrowardHandlerFunc() http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Info("get request", zap.String("url", request.URL.String()))
		api.reserveProxy.ServeHTTP(writer, request)
	}
}
