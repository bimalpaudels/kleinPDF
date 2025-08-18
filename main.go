package main

import (
	"embed"

	"kleinpdf/internal/app"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	application := app.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "KleinPDF",
		Width:  800,
		Height: 600,

		AssetServer: &assetserver.Options{
			Assets: assets,
		},

		OnStartup: application.OnStartup,
		Bind: []interface{}{
			application,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
