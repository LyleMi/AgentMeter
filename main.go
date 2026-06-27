package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"AgentMeter/internal/app"
	"AgentMeter/internal/startup"
	"AgentMeter/internal/tui"
)

func main() {
	var uiMode string
	var httpAddr string
	var staticDir string
	var start bool
	var skipBrowser bool
	var forceBuild bool
	flag.StringVar(&uiMode, "ui", "web", "UI mode: web or tui")
	flag.StringVar(&httpAddr, "http", "127.0.0.1:34115", "HTTP listen address, for example 127.0.0.1:34115")
	flag.StringVar(&staticDir, "static", "frontend/dist", "directory containing the built frontend assets")
	flag.BoolVar(&start, "start", false, "install/build frontend assets before starting web mode and open the browser")
	flag.BoolVar(&skipBrowser, "skip-browser", false, "with -start, do not open the browser")
	flag.BoolVar(&forceBuild, "force-build", false, "with -start, rebuild the frontend even when built assets look current")
	flag.Parse()

	uiMode = strings.ToLower(strings.TrimSpace(uiMode))
	if uiMode == "" {
		uiMode = "web"
	}

	if strings.HasPrefix(httpAddr, ":") {
		httpAddr = "127.0.0.1" + httpAddr
	}

	if (skipBrowser || forceBuild) && !start {
		log.Fatal("-skip-browser and -force-build require -start")
	}
	if start {
		if uiMode != "web" {
			log.Fatal("-start can only be used with -ui web")
		}
		repoRoot, err := startup.FindRepoRoot(".")
		if err != nil {
			log.Fatalf("find repository root: %v", err)
		}
		if !filepath.IsAbs(staticDir) {
			staticDir = filepath.Join(repoRoot, staticDir)
		}
		if err := startup.EnsureWebAssets(repoRoot, forceBuild); err != nil {
			log.Fatalf("prepare frontend: %v", err)
		}
	}

	service, err := app.New()
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if err := service.Startup(context.Background()); err != nil {
		log.Fatalf("startup: %v", err)
	}

	switch uiMode {
	case "web":
		if start && !skipBrowser {
			startup.OpenBrowserAfterDelay("http://"+httpAddr, 2*time.Second)
		}
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
