package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
)

type MediaServer struct {
	config   MediaServerConfig
	registry registry.Registry
}

type MediaServerConfig struct {
	Port int
}
