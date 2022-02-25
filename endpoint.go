package main

import (
	"encoding/json"
	"github.com/pingcap/log"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

// ReportEndpoint preprocess the request body and query param
type ReportEndpoint struct {
	http.HandlerFunc
}

func (ep *ReportEndpoint) QueryNodeGraph(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &QueryNodeGraphParam{}
		param.TiDBClusterID = req.URL.Query().Get("tidb_cluster_id")
		param.StartTS, _ = strconv.ParseInt(req.URL.Query().Get("start_ts"), 10, 64)
		param.EndTS, _ = strconv.ParseInt(req.URL.Query().Get("end_ts"), 10, 64)

		if err := param.Validate(); err != nil {
			log.Error("param validate failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}

		data, err := api.QueryNodeGraph(req.Context(), param)
		if err != nil {
			log.Error("query node graph failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		bs, err := json.Marshal(data)
		if err != nil {
			log.Error("json marshal failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		_, _ = w.Write(bs)
	}
}

func (ep *ReportEndpoint) QueryNodeGraphV2(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &QueryNodeGraphParam{}
		param.TiDBClusterID = req.URL.Query().Get("tidb_cluster_id")
		param.StartTS, _ = strconv.ParseInt(req.URL.Query().Get("start_ts"), 10, 64)
		param.EndTS, _ = strconv.ParseInt(req.URL.Query().Get("end_ts"), 10, 64)

		if err := param.Validate(); err != nil {
			log.Error("param validate failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}

		log.Info("QueryNodeGraphV2", zap.Any("param", param))
		data, err := api.QueryNodeGraphV2(req.Context(), param)
		if err != nil {
			log.Error("query node graph failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		bs, err := json.Marshal(data)
		if err != nil {
			log.Error("json marshal failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		_, _ = w.Write(bs)
	}
}

func (ep *ReportEndpoint) QueryAnnotation(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &QueryAnnotationsParam{}
		param.TiDBClusterID = req.URL.Query().Get("tidb_cluster_id")
		param.StartTS, _ = strconv.ParseInt(req.URL.Query().Get("start_ts"), 10, 64)
		param.EndTS, _ = strconv.ParseInt(req.URL.Query().Get("end_ts"), 10, 64)
		param.Measurement = req.URL.Query().Get("measurement")
		// log.Info("QueryAnnotation", zap.Any("param", param))
		data, err := api.QueryAnnotations(req.Context(), param)
		if err != nil {
			log.Error("query annotations failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusInternalServerError)
			return
		}
		ResponseWithJSON(w, data)
	}
}

func (ep *ReportEndpoint) QueryAnnotationV2(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &QueryAnnotationsParam{}
		param.TiDBClusterID = req.URL.Query().Get("tidb_cluster_id")
		param.StartTS, _ = strconv.ParseInt(req.URL.Query().Get("start_ts"), 10, 64)
		param.EndTS, _ = strconv.ParseInt(req.URL.Query().Get("end_ts"), 10, 64)
		param.Measurement = req.URL.Query().Get("measurement")
		log.Info("QueryAnnotationV2", zap.Any("param", param))
		data, err := api.QueryAnnotationsV2(req.Context(), param)
		if err != nil {
			log.Error("query annotations failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusInternalServerError)
			return
		}
		ResponseWithJSON(w, data)
	}
}

func (ep *ReportEndpoint) InsertSample(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &InsertSampleParam{
			Fields: make(map[string]interface{}),
			Tags:   make(map[string]string),
		}
		if err := json.NewDecoder(req.Body).Decode(param); err != nil {
			log.Error("json marshal failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		if err := param.Validate(); err != nil {
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}

		data, err := api.InsertSample(req.Context(), param)
		if err != nil {
			log.Error("insert sample failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusInternalServerError)
			return
		}
		ResponseWithJSON(w, data)
	}
}

func (ep *ReportEndpoint) Flush(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if err := api.Flush(req.Context()); err != nil {
			log.Error("flush failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusInternalServerError)
			return
		}
		ResponseWithJSON(w, struct{}{})
	}
}

func (ep *ReportEndpoint) InsertSampleV2(api *ReportAPI) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		param := &InsertSampleParam{
			Fields: make(map[string]interface{}),
			Tags:   make(map[string]string),
		}
		if err := json.NewDecoder(req.Body).Decode(param); err != nil {
			log.Error("json marshal failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}
		if err := param.Validate(); err != nil {
			ResponseWithStatus(w, http.StatusBadRequest)
			return
		}

		log.Info("InsertSampleV2", zap.Any("param", param))
		data, err := api.InsertSampleV2(req.Context(), param)
		if err != nil {
			log.Error("insert sample failed", zap.Error(err))
			ResponseWithStatus(w, http.StatusInternalServerError)
			return
		}
		ResponseWithJSON(w, data)
	}
}
func ResponseWithStatus(w http.ResponseWriter, statusCode int) {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte("{}"))
}

func ResponseWithJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Add("Content-type", "application/json")
	bs, err := json.Marshal(data)
	if err != nil {
		log.Error("json marshal failed", zap.Error(err))
		ResponseWithStatus(w, http.StatusInternalServerError)
		return
	}
	_, _ = w.Write(bs)
}
