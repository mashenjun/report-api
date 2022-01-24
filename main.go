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

	api, err := NewReportAPI(cfg.InfluxDB.Endpoint, cfg.InfluxDB.Org, cfg.InfluxDB.Bucket, cfg.InfluxDB.Token)
	if err != nil {
		log.Fatalln(err)
	}
	defer api.Close()
	ep := ReportEndpoint{}
	// construct  router
	router := mux.NewRouter()
	router.HandleFunc("/node_graph", ep.QueryNodeGraph(api)).Methods(http.MethodGet)
	router.HandleFunc("/annotations", ep.QueryAnnotation(api)).Methods(http.MethodPost)
	router.HandleFunc("/sample", ep.InsertSample(api)).Methods(http.MethodPost)
	router.HandleFunc("/flush", ep.Flush(api)).Methods(http.MethodPost)
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
	log.Println("shutting down...")
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}
