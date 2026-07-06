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

type runtimeConfig struct {
	uiMode      string
	httpAddr    string
	staticDir   string
	start       bool
	skipBrowser bool
	forceBuild  bool
}

func main() {
	if exitCode, ok := runCLICommand(os.Args[1:]); ok {
		os.Exit(exitCode)
	}

	config := parseRuntimeConfig(os.Args[1:])
	service := newStartedApp()
	runConfiguredUI(config, service)
}

func runCLICommand(args []string) (int, bool) {
	if len(args) == 0 || !cli.IsCommand(args[0]) {
		return 0, false
	}
	return cli.Run(args, os.Stdout, os.Stderr), true
}

func parseRuntimeConfig(args []string) runtimeConfig {
	config := runtimeConfig{
		uiMode:    "web",
		httpAddr:  "127.0.0.1:34115",
		staticDir: "frontend/dist",
	}
	flag.StringVar(&config.uiMode, "ui", config.uiMode, "UI mode: web or tui")
	flag.StringVar(&config.httpAddr, "http", config.httpAddr, "HTTP listen address, for example 127.0.0.1:34115")
	flag.StringVar(&config.staticDir, "static", config.staticDir, "directory containing the built frontend assets")
	flag.BoolVar(&config.start, "start", false, "install/build frontend assets before starting web mode and open the browser")
	flag.BoolVar(&config.skipBrowser, "skip-browser", false, "with -start, do not open the browser")
	flag.BoolVar(&config.forceBuild, "force-build", false, "with -start, rebuild the frontend even when built assets look current")
	flag.Usage = func() {
		cli.PrintUsage(os.Stderr)
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}
	flag.CommandLine.Parse(normalizeCommandArgs(args))
	if flag.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unknown command or argument %q\n\n", flag.Arg(0))
		cli.PrintUsage(os.Stderr)
		os.Exit(cli.ExitUsage)
	}

	config.uiMode = strings.ToLower(strings.TrimSpace(config.uiMode))
	if config.uiMode == "" {
		config.uiMode = "web"
	}
	if strings.HasPrefix(config.httpAddr, ":") {
		config.httpAddr = "127.0.0.1" + config.httpAddr
	}
	if (config.skipBrowser || config.forceBuild) && !config.start {
		log.Fatal("-skip-browser and -force-build require -start")
	}
	if config.start {
		if config.uiMode != "web" {
			log.Fatal("-start can only be used with -ui web")
		}
		preparedStaticDir, err := startup.PrepareWebAssets(config.staticDir, config.forceBuild)
		if err != nil {
			log.Fatalf("prepare frontend: %v", err)
		}
		config.staticDir = preparedStaticDir
	}
	return config
}

func newStartedApp() *app.App {
	service, err := app.New()
	if err != nil {
		log.Fatalf("create app: %v", err)
	}
	if err := service.Startup(context.Background()); err != nil {
		log.Fatalf("startup: %v", err)
	}
	return service
}

func runConfiguredUI(config runtimeConfig, service *app.App) {
	switch config.uiMode {
	case "web":
		if config.start && !config.skipBrowser {
			startup.OpenBrowserAfterDelay("http://"+config.httpAddr, 2*time.Second)
		}
		startWeb(service, config.httpAddr, config.staticDir)
	case "tui":
		if err := tui.Run(context.Background(), service); err != nil {
			log.Fatalf("tui: %v", err)
		}
	default:
		log.Fatalf("unknown -ui mode %q; expected web or tui", config.uiMode)
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
