package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/LyleMi/AgentMeter/internal/app"
	"github.com/LyleMi/AgentMeter/internal/cli"
	"github.com/LyleMi/AgentMeter/internal/startup"
	"github.com/LyleMi/AgentMeter/internal/tui"
)

func main() {
	if len(os.Args) > 1 && cli.IsCommand(os.Args[1]) {
		os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
	}

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
	flag.Usage = func() {
		cli.PrintUsage(os.Stderr)
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}
	flag.CommandLine.Parse(normalizeCommandArgs(os.Args[1:]))
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unknown command or argument %q\n\n", flag.Arg(0))
		cli.PrintUsage(os.Stderr)
		os.Exit(cli.ExitUsage)
	}

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
		preparedStaticDir, err := startup.PrepareWebAssets(staticDir, forceBuild)
		if err != nil {
			log.Fatalf("prepare frontend: %v", err)
		}
		staticDir = preparedStaticDir
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

func normalizeCommandArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "tui", "cli":
		return prependArgs([]string{"-ui", "tui"}, args[1:])
	case "web":
		return prependArgs([]string{"-ui", "web"}, args[1:])
	case "start":
		return prependArgs([]string{"-start"}, args[1:])
	default:
		return args
	}
}

func prependArgs(prefix, rest []string) []string {
	normalized := make([]string, 0, len(prefix)+len(rest))
	normalized = append(normalized, prefix...)
	normalized = append(normalized, rest...)
	return normalized
}

func startWeb(service *app.App, httpAddr, staticDir string) {
	mux := http.NewServeMux()
	app.RegisterHTTPHandlers(mux, service, os.DirFS(staticDir))
	fmt.Fprintf(os.Stdout, "AgentMeter listening on http://%s\n", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, mux))
}
