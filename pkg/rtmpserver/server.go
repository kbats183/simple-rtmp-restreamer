package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/yapingcat/gomedia/go-rtmp"
	"log"
	"net"
	"strconv"
)

func prepareConfig(config MediaServerConfig) MediaServerConfig {
	if config.Port == 0 {
		config.Port = 1935
	}
	return config
}

func NewMediaServer(config MediaServerConfig, registry registry.Registry) *MediaServer {
	config = prepareConfig(config)
	return &MediaServer{
		config:   config,
		registry: registry,
	}
}

func (s *MediaServer) Start() {
	addr := "0.0.0.0:" + strconv.Itoa(s.config.Port)
	listen, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatalf("Failed to start RTMP server: %v", err)
	}
	for {
		conn, _ := listen.Accept()
		sess := s.newMediaSession(conn)
		sess.init()
		go sess.start()
	}
}

func (s *MediaServer) newMediaSession(conn net.Conn) *MediaSession {
	return &MediaSession{
		id:        genId(),
		conn:      conn,
		handle:    rtmp.NewRtmpServerHandle(),
		quit:      make(chan struct{}),
		frameCome: make(chan struct{}, 1),
		registry:  s.registry,
	}
}
