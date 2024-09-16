package main

import (
	"context"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/apiserver"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/rtmpserver"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	setupLogger()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	streamRegistry := registry.NewRegistry()
	log.Println("Starting...")

	rtmp := rtmpserver.NewMediaServer(rtmpserver.MediaServerConfig{}, streamRegistry)
	web := apiserver.NewWebServer(streamRegistry)

	go func() {
		if err := rtmp.Start(ctx); err != nil {
			log.Printf("RTMP server error: %v", err)
			cancel()
		}
	}()

	go func() {
		if err := web.Start(ctx); err != nil {
			log.Printf("API server error: %v", err)
			cancel()
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the servers
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down servers...")

	cancel()

	rtmp.Stop()

	if err := web.Stop(); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	log.Println("Servers stopped")
}
