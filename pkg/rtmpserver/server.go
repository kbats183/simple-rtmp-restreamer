package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/utils"
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
		sessions: make(map[string]*MediaSession),
	}
}

func (s *MediaServer) Start() {
	addr := "0.0.0.0:" + strconv.Itoa(s.config.Port)
	listen, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatalf("Failed to start RTMP server: %v", err)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		sess := s.newMediaSession(conn)
		s.mu.Lock()
		s.sessions[sess.id] = sess
		s.mu.Unlock()
		sess.init()
		go func() {
			sess.start()
			s.mu.Lock()
			delete(s.sessions, sess.id)
			s.mu.Unlock()
		}()
	}
}

func (s *MediaServer) newMediaSession(conn net.Conn) *MediaSession {
	return &MediaSession{
		id:       utils.GenId(),
		conn:     conn,
		handle:   rtmp.NewRtmpServerHandle(),
		quit:     make(chan struct{}),
		registry: s.registry,
	}
}
