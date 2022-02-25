package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	http2 "github.com/influxdata/influxdb-client-go/v2/api/http"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	lp "github.com/influxdata/line-protocol"
	"github.com/pingcap/log"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type ReportAPIOption func(reportAPI *ReportAPI) error

type ReportAPI struct {
	// vmEndpoint has following format <scheme>://host:[port]
	vmEndpoint string

	bucket string
	org    string

	influxCli  influxdb2.Client
	writeAPI   api.WriteAPI
	queryAPI   api.QueryAPI
	writeErrCh <-chan error

	httpCli http.Client

	// internal variable
	done chan struct{}
}

func WithVMOption(endpoint string) ReportAPIOption {
	return func(reportAPI *ReportAPI) error {
		reportAPI.vmEndpoint = endpoint
		return nil
	}
}
func NewReportAPI(influxdbEp string, org string, bucket string, token string, opts ...ReportAPIOption) (*ReportAPI, error) {
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

	rAPI.influxCli = influxdb2.NewClient(influxdbEp, token)
	rAPI.writeAPI = rAPI.influxCli.WriteAPI(org, bucket)
	rAPI.writeAPI.SetWriteFailedCallback(retryCallBack)
	rAPI.writeErrCh = rAPI.writeAPI.Errors()
	rAPI.queryAPI = rAPI.influxCli.QueryAPI(org)
	// TODO(shenjun): use cutomized transport later
	rAPI.httpCli = http.Client{
		Transport: http.DefaultTransport,
	}

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
	// fmt.Println(fluxQuery)
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
	// log.Info("", zap.Any("nodesLookuo", nodesLookup))
	for _, node := range data.Nodes {
		id, _ := strconv.ParseInt(node.ID, 0, 64)
		if targets, ok := EdgeMatrixV2[id]; ok {
			// log.Info("", zap.Any("targets", targets), zap.Any("id", id))
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
	|> filter(fn:(r) => r._measurement =="%s" and r.tidb_cluster_id =="%v")
`
	if len(param.Measurement) == 0 {
		param.Measurement = "fast_tune_anomaly"
	}
	fluxQuery := fmt.Sprintf(fluxQueryBase, api.bucket, param.StartTS, param.EndTS, param.Measurement, param.TiDBClusterID)

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
			Title:      "anomaly title",
			Tags:       "anomaly tags",
			Text:       "anomaly text",
		}
		rd := result.Record()
		// Time should be milliseconds
		item.Time = rd.Time().UnixNano() / 1e6
		if rd.Field() == "end_time" {
			if endTs, ok := rd.Value().(float64); ok {
				item.TimeEnd = int64(endTs) * 1e3
			}
		}
		if panelID, ok := rd.ValueByKey("panel_id").(string); ok {
			item.PanelID, _ = strconv.ParseInt(panelID, 0, 64)
		}
		if title, ok := rd.ValueByKey("title").(string); ok {
			item.Title = title
		}
		if tags, ok := rd.ValueByKey("tags").(string); ok {
			item.Tags = tags
		}
		if text, ok := rd.ValueByKey("text").(string); ok {
			item.Text = text
		}
		data = append(data, item)
	}

	if result.Err() != nil {
		log.Error("query parsing failed", zap.Error(result.Err()))
		return nil, err
	}
	return data, nil
}

func (api *ReportAPI) QueryDynamicTextValue(ctx context.Context, param *QueryDynamicTextValueParam) (QueryDynamicTextValueData, error) {
	fluxQueryBase := `
from(bucket: "%s")
	|> range(start: %v, stop: %v)
	|> filter(fn:(r) => r._measurement =="%s" and r.tidb_cluster_id == "%v")
	|> group(columns: ["_field"])
	|> first()
`
	if len(param.Measurement) == 0 {
		param.Measurement = "diagnosis_overview"
	}
	fluxQuery := fmt.Sprintf(fluxQueryBase, api.bucket, param.StartTS, param.EndTS, param.Measurement, param.TiDBClusterID)

	result, err := api.queryAPI.Query(ctx, fluxQuery)
	if err != nil {
		log.Error("query influxdb failed", zap.Error(err))
		return nil, err
	}
	defer result.Close()

	data := make(QueryDynamicTextValueData)
	for result.Next() {
		rd := result.Record()
		value, ok := rd.Value().(float64)
		if !ok {
			continue
		}
		switch rd.ValueByKey("format") {
		case "float":
			data[rd.Field()] = value
		case "int":
			data[rd.Field()] = int64(value)
		case "unix_seconds":
			data[rd.Field()] = int64(value)
			data[fmt.Sprintf("%s_rfc3339", rd.Field())] = time.Unix(int64(value), 0).Format(time.RFC3339)
		default:
			data[rd.Field()] = value
		}
	}

	if result.Err() != nil {
		log.Error("query parsing failed", zap.Error(result.Err()))
		return nil, err
	}
	return data, nil
}

// use `/api/v1/query` to get raw sample
func (api *ReportAPI) queryMetrics(ctx context.Context, queryExpr string, ts int64) (model.Value, error) {
	u := fmt.Sprintf("%s%s", api.vmEndpoint, "/api/v1/query")
	payload := url.Values{
		"query": {queryExpr},
		"time":  {strconv.FormatInt(ts, 10)},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(payload.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := api.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, err
	}

	mResp := MetricsResp{}
	if err := json.NewDecoder(resp.Body).Decode(&mResp); err != nil {
		return nil, err
	}
	// matrix, ok := mResp.Data.v.(model.Matrix)
	// if !ok {
	// 	return nil, fmt.Errorf("type %T not support", mResp.Data.v)
	// }
	return mResp.Data.v, nil
}

func (api *ReportAPI) QueryNodeGraphV2(ctx context.Context, param *QueryNodeGraphParam) (*QueryNodeGraphData, error) {
	ts, interval := param.GetRollUpParam()
	queryExpr := fmt.Sprintf(`first_over_time({__name__=~"fast_tune_similarity.*",tidb_cluster_id="%s"}[%s])`, param.TiDBClusterID, interval)
	v, err := api.queryMetrics(ctx, queryExpr, ts)
	if err != nil {
		return nil, err
	}
	vector, ok := v.(model.Vector)
	if !ok {
		log.Error("convert to vector failed", zap.Any("value", v))
		return nil, fmt.Errorf("")
	}
	log.Info("QueryNodeGraphV2", zap.Int("len", len(vector)))
	data := QueryNodeGraphData{
		Nodes: make([]*Node, 0),
		Edges: make([]*Edge, 0),
	}

	nodesLookup := make(map[int64]struct{})
	if len(vector) == 0 {
		return &data, nil
	}
	for _, sample := range vector {
		similarity := float64(sample.Value)
		idStr := string(sample.Metric["id"])
		id, err := strconv.ParseInt(idStr, 0, 64)
		if err != nil {
			log.Warn("parse int fail skip the node", zap.String("id", idStr), zap.Error(err))
			continue
		}
		node := DefaultNode()

		node.ID = idStr
		node.Title = idStr
		node.SubTitle = string(sample.Metric["title"])
		node.MainStat = fmt.Sprintf("%.3f", similarity)
		node.ArcPositive = similarity
		node.ArcNegative = 1 - similarity

		data.Nodes = append(data.Nodes, node)
		nodesLookup[id] = struct{}{}
	}
	for _, node := range data.Nodes {
		id, _ := strconv.ParseInt(node.ID, 0, 64)
		if targets, ok := EdgeMatrixV2[id]; ok {
			for _, target := range targets {
				if _, ok := nodesLookup[target]; !ok {
					continue
				}
				edge := DefaultEdge()
				edge.ID = fmt.Sprintf("%v%v", id, target)
				edge.Source = node.ID
				edge.Target = fmt.Sprintf("%v", target)
				data.Edges = append(data.Edges, edge)
			}
		}
	}

	return &data, nil
}

func (api *ReportAPI) QueryAnnotationsV2(ctx context.Context, param *QueryAnnotationsParam) (QueryAnnotationsData, error) {
	if len(param.Measurement) == 0 {
		param.Measurement = "fast_tune_anomaly"
	}
	ts, interval := param.GetRollUpParam()
	queryExpr := fmt.Sprintf(`{__name__=~"%s.*",tidb_cluster_id="%s"}[%s]`, param.Measurement, param.TiDBClusterID, interval)
	v, err := api.queryMetrics(ctx, queryExpr, ts)
	if err != nil {
		return nil, err
	}
	matrix, ok := v.(model.Matrix)
	if !ok {
		log.Error("convert to matrix failed", zap.Any("value", v))
		return nil, fmt.Errorf("")
	}
	data := make(QueryAnnotationsData, 0)
	for _, sample := range matrix {
		for _, pair := range sample.Values {
			item := QueryAnnotationItem{
				Annotation: DefaultAnomalyAnnotation(),
				Title:      "anomaly title",
				Tags:       "anomaly tags",
				Text:       "anomaly text",
			}
			if panelID, ok := sample.Metric["panel_id"]; ok {
				item.PanelID, _ = strconv.ParseInt(string(panelID), 0, 64)
			}
			item.Title = string(sample.Metric["title"])
			item.Tags = string(sample.Metric["tags"])
			item.Text = string(sample.Metric["text"])
			item.Time = pair.Timestamp.UnixNano() / 1e6
			item.TimeEnd = int64(pair.Value) * 1e3
			data = append(data, item)
		}
	}
	log.Info("QueryAnnotationsV2", zap.Int("len", len(data)))

	return data, nil
}

func (api *ReportAPI) QueryDynamicTextValueV2(ctx context.Context, param *QueryDynamicTextValueParam) (QueryDynamicTextValueData, error) {
	ts, interval := param.GetRollUpParam()
	queryExpr := fmt.Sprintf(`first_over_time({__name__=~"%s.*",tidb_cluster_id="%s"}[%s])`, param.Measurement, param.TiDBClusterID, interval)
	v, err := api.queryMetrics(ctx, queryExpr, ts)
	if err != nil {
		return nil, err
	}
	vector, ok := v.(model.Vector)
	if !ok {
		log.Error("convert to vector failed", zap.Any("value", v))
		return nil, fmt.Errorf("")
	}
	data := make(QueryDynamicTextValueData)
	if len(vector) == 0 {
		return data, nil
	}
	for _, sample := range vector {
		value := float64(sample.Value)
		metricsName := string(sample.Metric["__name__"])
		key := strings.TrimPrefix(metricsName, param.Measurement)
		data[key] = value
		switch sample.Metric["format"] {
		case "float":
			data[key] = value
		case "int":
			data[key] = int64(value)
		case "unix_seconds":
			data[key] = int64(value)
			data[fmt.Sprintf("%s_rfc3339", key)] = time.Unix(int64(value), 0).Format(time.RFC3339)
		default:
			data[key] = value
		}
	}

	return data, nil
}

// TODO(shenjun): how to handle the error with async write?
// InsertSample insert time series data in to influxdb
func (api *ReportAPI) InsertSample(ctx context.Context, param *InsertSampleParam) (*InsertSampleData, error) {
	log.Info("InsertSample", zap.Any("param", param))
	ts := time.Unix(param.Timestamp, 0)
	point := influxdb2.NewPoint(param.Measurement, param.GetTags(), param.Fields, ts)
	api.writeAPI.WritePoint(point)
	return &InsertSampleData{}, nil
}

// InsertSampleV2 insert time series data in to victoria metrics no need to call flush
// the metrics will be saved as <measurement>_<field_name> and value will be <field_value>
func (api *ReportAPI) InsertSampleV2(ctx context.Context, param *InsertSampleParam) (*InsertSampleData, error) {
	ts := time.Unix(param.Timestamp, 0)
	point := influxdb2.NewPoint(param.Measurement, param.GetTags(), param.Fields, ts)
	payload, err := encodePoints(point)
	if err != nil {
		return nil, err
	}
	log.Info("InsertSampleV2", zap.String("payload", payload))
	u := fmt.Sprintf("%s%s", api.vmEndpoint, "/influx/api/v2/write")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, strings.NewReader(payload))
	if err != nil {
		log.Error("new request failed", zap.Error(err))
		return nil, err
	}
	resp, err := api.httpCli.Do(req)
	if err != nil {
		log.Error("do request failed", zap.Error(err))
		return nil, err
	}
	if resp.StatusCode/100 != 2 {
		log.Error("response is not ok", zap.String("status", resp.Status))
		return nil, fmt.Errorf("response status is %v", resp.StatusCode)
	}
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

func encodePoints(point *write.Point) (string, error) {
	var buffer bytes.Buffer
	e := lp.NewEncoder(&buffer)
	e.SetFieldTypeSupport(lp.UintSupport)
	e.FailOnFieldErr(true)
	e.SetPrecision(time.Nanosecond)
	_, err := e.Encode(point)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}
