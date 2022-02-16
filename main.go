package main

import (
	"context"
	"flag"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var (
	cfgPath string
)

func main() {
	flag.StringVar(&cfgPath, "c", "./", "reportd -c=/path/to/config.yaml")
	flag.Parse()

	cfg, err := InitConfig(cfgPath)
	if err != nil {
		log.Fatalln(err)
	}

	dataAPI, err := NewDataAPI(cfg.VM.Endpoint)
	if err != nil {
		log.Fatal(err)
	}
	reportAPI, err := NewReportAPI(
		cfg.InfluxDB.Endpoint, cfg.InfluxDB.Org, cfg.InfluxDB.Bucket, cfg.InfluxDB.Token,
		WithVMOption(cfg.VM.Endpoint))
	if err != nil {
		log.Fatalln(err)
	}
	defer reportAPI.Close()
	ep := ReportEndpoint{}
	// construct  router
	router := mux.NewRouter()
	// report api
	router.HandleFunc("/node_graph", ep.QueryNodeGraph(reportAPI)).Methods(http.MethodGet)
	router.HandleFunc("/node_graph/v2", ep.QueryNodeGraphV2(reportAPI)).Methods(http.MethodGet)
	router.HandleFunc("/annotations", ep.QueryAnnotation(reportAPI)).Methods(http.MethodGet)
	router.HandleFunc("/annotations/v2", ep.QueryAnnotationV2(reportAPI)).Methods(http.MethodGet)
	router.HandleFunc("/sample", ep.InsertSample(reportAPI)).Methods(http.MethodPost)
	router.HandleFunc("/sample/v2", ep.InsertSampleV2(reportAPI)).Methods(http.MethodPost)
	router.HandleFunc("/flush", ep.Flush(reportAPI)).Methods(http.MethodPost)
	// data api just forward request to vm
	router.HandleFunc("/data/metrics", dataAPI.GetMetricsFrowardHandlerFunc()).Methods(http.MethodGet)
	// construct http server
	httpServer := &http.Server{
		Addr:    ":8081",
		Handler: router,
	}
	go func() {
		log.Printf("start listen and serve on %s\n", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	<-ctx.Done()
	defer stop()
	// graceful shutdown the http server
	log.Println("shutting down ...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
