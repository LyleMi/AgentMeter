package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"AgentMeter/internal/app"
)

func main() {
	var httpAddr string
	var staticDir string
	flag.StringVar(&httpAddr, "http", "127.0.0.1:34115", "HTTP listen address, for example 127.0.0.1:34115")
	flag.StringVar(&staticDir, "static", "frontend/dist", "directory containing the built frontend assets")
	flag.Parse()

	service, err := app.New()
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if strings.HasPrefix(httpAddr, ":") {
		httpAddr = "127.0.0.1" + httpAddr
	}

	if err := service.Startup(context.Background()); err != nil {
		log.Fatalf("startup: %v", err)
	}
	mux := http.NewServeMux()
	app.RegisterHTTPHandlers(mux, service, os.DirFS(staticDir))
	fmt.Fprintf(os.Stdout, "AgentMeter listening on http://%s\n", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, mux))
}
