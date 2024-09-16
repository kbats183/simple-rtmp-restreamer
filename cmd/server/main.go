package main

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/apiserver"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/rtmpserver"
)

func main() {
	setupLogger()

	streamRegistry := registry.NewRegistry()
	println("Starting...")

	rtmp := rtmpserver.NewMediaServer(rtmpserver.MediaServerConfig{}, streamRegistry)
	web := apiserver.NewWebServer(streamRegistry)
	go rtmp.Start()
	web.Start()
}
