package main

import (
	"embed"

	"kleinpdf/internal/application"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := application.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "KleinPDF",
		Width:  800,
		Height: 600,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		OnStartup: app.OnStartup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
