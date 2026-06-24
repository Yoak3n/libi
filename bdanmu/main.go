package main

import (
	"embed"
	"log"

	"bdanmu/app/dispatch"
	"bdanmu/app/runtime/cache"
	"bdanmu/app/service"

	"github.com/Yoak3n/libi/shared/config"
	"github.com/Yoak3n/libi/shared/database"
	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[string](service.EventQRCode)
	application.RegisterEvent[bool](service.EventLoginResult)
	application.RegisterEvent[*dispatch.Message]("message")
	application.RegisterEvent[string]("started")
	application.RegisterEvent[string]("error")
	application.RegisterEvent[int]("change")
}

func main() {
	if config.Conf.Database != nil && config.Conf.Database.Type != "" {
		database.InitDatabase()
		cache.Init(database.DB())
	}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	authService := &service.AuthService{}

	app := application.New(application.Options{
		Name:        "bdanmu",
		Description: "Bilibili live room danmu monitor",
		Services: []application.Service{
			application.NewService(authService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
		Windows: application.WindowsOptions{
			WebviewUserDataPath: "data/webview",
		},
	})

	authService.Emitter = app.Event

	// 初始化 Dispatcher：消息转发层
	dispatcher := dispatch.NewDispatcher(app.Event, database.DB(), cache.GetUserInfoMultiply)
	authService.Dispatcher = dispatcher

	liveRoom := &service.LiveRoom{Emitter: app.Event, Dispatcher: dispatcher}

	// 监听前端切换房间事件
	app.Event.On(service.EventChange, func(ev *application.CustomEvent) {
		switch v := ev.Data.(type) {
		case float64:
			config.Conf.RoomId = int(v)
			liveRoom.ConnectRoom(int(v))
		case int:
			config.Conf.RoomId = v
			liveRoom.ConnectRoom(v)
		}
	})

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "",
		Width:            512,
		Height:           900,
		Frameless:        true,
		MinWidth:         512,
		MaxWidth:         512,
		BackgroundColour: application.NewRGBA(28, 28, 28, 255),
		URL:              "/",
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: false,
		},
	})

	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
