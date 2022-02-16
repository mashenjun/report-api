package main

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/prometheus/common/model"
)

type QueryNodeGraphParam struct {
	TsRange
	// StartTS       int64  `json:"start_ts"`
	// EndTS         int64  `json:"end_ts"`
	TiDBClusterID string `json:"tidb_cluster_id"`
}

func (param *QueryNodeGraphParam) GetRollUpParam() (int64, string) {
	return param.TsRange.GetRollUpParam()
	// return param.EndTS, fmt.Sprintf("%vs", param.EndTS-param.StartTS)
}

func (param *QueryNodeGraphParam) Validate() error {
	if param.StartTS == 0 {
		return errors.New("start_ts is zero")
	}
	if param.EndTS == 0 {
		return errors.New("end_ts is zero")
	}
	if len(param.TiDBClusterID) == 0 {
		return errors.New("tidb_cluster_id is zero")
	}
	return nil
}

type Node struct {
	ID            string `json:"id"`
	Title         string `json:"title"`
	SubTitle      string `json:"subTitle"`
	MainStat      string `json:"mainStat"`
	SecondaryStat string `json:"secondaryStat"`
	// TODO(shenjun): rename following fields
	ArcPositive      float64 `json:"arc__similarity"`
	ArcNegative      float64 `json:"arc__nusimilarity"`
	ArcPositiveColor string  `json:"arc__similarity_color"`
	ArcNegativeColor string  `json:"arc__nusimilarity_color"`
}

func DefaultNode() *Node {
	return &Node{
		ArcPositiveColor: "red",
		ArcNegativeColor: "green",
	}
}

type Edge struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	Target        string `json:"target"`
	MainStat      string `json:"mainStat,omitempty"`
	SecondaryStat string `json:"secondaryStat,omitempty"`
}

func DefaultEdge() *Edge {
	return &Edge{}
}

type QueryNodeGraphData struct {
	Nodes []*Node `json:"nodes"`
	Edges []*Edge `json:"edges"`
}

type TsRange struct {
	StartTS int64 `json:"start_ts"`
	EndTS   int64 `json:"end_ts"`
}

// GetRollUpParam retunr timestamp in unix and the window size
func (tr *TsRange) GetRollUpParam() (int64, string) {
	return tr.EndTS, fmt.Sprintf("%vs", tr.EndTS-tr.StartTS)
}

type QueryAnnotationsParam struct {
	TsRange
	// StartTS       int64  `json:"start_ts"`
	// EndTS         int64  `json:"end_ts"`
	TiDBClusterID string `json:"tidb_cluster_id"`
	Measurement   string `json:"measurement"`
}

// GetRollUpParam retunr timestamp in unix and the window size
func (param *QueryAnnotationsParam) GetRollUpParam() (int64, string) {
	return param.TsRange.GetRollUpParam()
	// return param.EndTS, fmt.Sprintf("%vs", param.EndTS-param.StartTS)
}

func (param *QueryAnnotationsParam) Validate() error {
	if param.StartTS == 0 {
		return errors.New("start_ts is zero")
	}
	if param.EndTS == 0 {
		return errors.New("end_ts is zero")
	}
	if len(param.TiDBClusterID) == 0 {
		return errors.New("tidb_cluster_id is zero")
	}
	return nil
}

type Annotation struct {
	Name       string `json:"name"`
	Datasource string `json:"datasource"`
	IconColor  string `json:"iconColor"`
	Enable     bool   `json:"enable"`
	ShowLine   bool   `json:"showLine"`
	Query      string `json:"query"`
}

type QueryAnnotationItem struct {
	Annotation *Annotation `json:"annotation"`
	Time       int64       `json:"time"`
	TimeEnd    int64       `json:"timeEnd,omitempty"`
	Title      string      `json:"title"`
	Tags       string      `json:"tags"`
	Text       string      `json:"text"`
	PanelID    int64       `json:"panelId"`
}

func DefaultAnomalyAnnotation() *Annotation {
	return &Annotation{
		Name:       "Anomaly Point",
		Datasource: "Clinic",
		IconColor:  "rgba(255, 96, 96, 1)",
		Enable:     true,
		ShowLine:   true,
		Query:      "",
	}
}

type QueryAnnotationsData = []QueryAnnotationItem

// TODO(shenjun): define fields
type InsertSampleParam struct {
	Timestamp     int64                  `json:"timestamp"`
	Measurement   string                 `json:"measurement"`
	TiDBClusterID string                 `json:"tidb_cluster_id"`
	Fields        map[string]interface{} `json:"fields"`
	Tags          map[string]string      `json:"tags"`
}

func (param *InsertSampleParam) Validate() error {
	if param.Timestamp == 0 {
		return errors.New("timestamp is empty")
	}
	if len(param.Measurement) == 0 {
		return errors.New("measurement is empty")
	}
	if len(param.TiDBClusterID) == 0 {
		return errors.New("tidb_cluster_id is empty")
	}
	return nil
}

func (param *InsertSampleParam) GetTags() map[string]string {
	param.Tags["tidb_cluster_id"] = param.TiDBClusterID
	return param.Tags
}

// TODO(shenjun): define fields
type InsertSampleData struct{}

// type GetMetricsParam struct {
// 	Query string `yaml:"query"`
// 	Start int64  `yaml:"start"`
// 	End   int64  `yaml:"end"`
// 	Step  int64  `yaml:"step"`
// }

// copy from prometheus client golang.

type MetricsResp struct {
	Status string              `json:"status"`
	Data   *MetricsQueryResult `json:"data"`
}

type MetricsQueryResult struct {
	// The decoded value.
	v model.Value
}

func (qr *MetricsQueryResult) ToMatrix() (model.Matrix, error) {
	matrix, ok := qr.v.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("type %t is not model.matrix", qr.v)
	}
	return matrix, nil
}

func (qr *MetricsQueryResult) UnmarshalJSON(b []byte) error {
	v := struct {
		Type   model.ValueType `json:"resultType"`
		Result json.RawMessage `json:"result"`
	}{}

	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v.Type {
	case model.ValScalar:
		var sv model.Scalar
		err = json.Unmarshal(v.Result, &sv)
		qr.v = &sv

	case model.ValVector:
		var vv model.Vector
		err = json.Unmarshal(v.Result, &vv)
		qr.v = vv

	case model.ValMatrix:
		var mv model.Matrix
		err = json.Unmarshal(v.Result, &mv)
		qr.v = mv

	default:
		err = fmt.Errorf("unexpected value type %q", v.Type)
	}
	return err
}
