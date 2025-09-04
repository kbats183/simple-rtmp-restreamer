package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"sync"
)

type MediaServer struct {
	config   MediaServerConfig
	registry registry.Registry
	sessions map[string]*MediaSession
	mu       sync.Mutex
}

type MediaServerConfig struct {
	Port int
}
