package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"AgentMeter/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	var httpAddr string
	flag.StringVar(&httpAddr, "http", "", "serve AgentMeter as a local HTTP app instead of launching Wails, for example :34115")
	flag.Parse()

	service, err := app.New()
	if err != nil {
		log.Fatalf("create app: %v", err)
	}

	if httpAddr != "" {
		if err := service.Startup(context.Background()); err != nil {
			log.Fatalf("startup: %v", err)
		}
		mux := http.NewServeMux()
		app.RegisterHTTPHandlers(mux, service, assets)
		fmt.Fprintf(os.Stdout, "AgentMeter listening on http://127.0.0.1%s\n", httpAddr)
		log.Fatal(http.ListenAndServe(httpAddr, mux))
	}

	err = wails.Run(&options.App{
		Title:  "AgentMeter",
		Width:  1280,
		Height: 820,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 246, G: 248, B: 251, A: 1},
		OnStartup: func(ctx context.Context) {
			if err := service.Startup(ctx); err != nil {
				log.Printf("startup: %v", err)
			}
		},
		Bind: []interface{}{
			service,
		},
	})
	if err != nil {
		log.Fatalf("run: %v", err)
	}
}
