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
	"AgentMeter/internal/tui"
)

func main() {
	var uiMode string
	var httpAddr string
	var staticDir string
	flag.StringVar(&uiMode, "ui", "web", "UI mode: web or tui")
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

	switch strings.ToLower(strings.TrimSpace(uiMode)) {
	case "", "web":
		startWeb(service, httpAddr, staticDir)
	case "tui":
		if err := tui.Run(context.Background(), service); err != nil {
			log.Fatalf("tui: %v", err)
		}
	default:
		log.Fatalf("unknown -ui mode %q; expected web or tui", uiMode)
	}
}

func startWeb(service *app.App, httpAddr, staticDir string) {
	mux := http.NewServeMux()
	app.RegisterHTTPHandlers(mux, service, os.DirFS(staticDir))
	fmt.Fprintf(os.Stdout, "AgentMeter listening on http://%s\n", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, mux))
}
