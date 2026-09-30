package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := newApp()
	err := wails.Run(&options.App{
		Title: "pomodoro", Width: 440, Height: 640,
		DisableResize: true, Frameless: true,
		BackgroundColour: options.NewRGB(248, 247, 244),
		AssetServer:      &assetserver.Options{Assets: assets},
		OnStartup:        app.startup, OnShutdown: app.shutdown,
		Windows:            &windows.Options{OnSuspend: app.suspend, OnResume: app.resume},
		SingleInstanceLock: &options.SingleInstanceLock{UniqueId: "a30c822e-98aa-4d09-b8ee-886710d2af15"},
		Bind:               []interface{}{app},
	})
	if err != nil {
		log.Fatal(err)
	}
}
