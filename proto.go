package main

import "errors"

// TODO(shenjun): define fields
type QueryNodeGraphParam struct {
	StartTS       int64  `json:"start_ts"`
	EndTS         int64  `json:"end_ts"`
	TiDBClusterID string `json:"tidb_cluster_id"`
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

// TODO(shenjun): define fields
type QueryAnnotationsParam struct {
	StartTS       int64  `json:"start_ts"`
	EndTS         int64  `json:"end_ts"`
	TiDBClusterID string `json:"tidb_cluster_id"`
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
type InsertSampleData struct {
}

type GetMetricsParam struct {
	Query string `yaml:"query"`
	Start int64  `yaml:"start"`
	End   int64  `yaml:"end"`
	Step  int64  `yaml:"step"`
}
