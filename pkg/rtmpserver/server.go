package rtmpserver

import (
	"context"
	"errors"
	"fmt"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/yapingcat/gomedia/go-rtmp"
	"log"
	"net"
	"strconv"
	"sync"
)

func prepareConfig(config MediaServerConfig) MediaServerConfig {
	if config.Port == 0 {
		config.Port = 1935
	}
	return config
}

type MediaServer struct {
	config   MediaServerConfig
	registry registry.Registry
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

func NewMediaServer(config MediaServerConfig, registry registry.Registry) *MediaServer {
	config = prepareConfig(config)
	ctx, cancel := context.WithCancel(context.Background())
	return &MediaServer{
		config:   config,
		registry: registry,
		ctx:      ctx,
		cancel:   cancel,
		wg:       sync.WaitGroup{},
	}
}

func (s *MediaServer) Start(ctx context.Context) error {
	addr := "0.0.0.0:" + strconv.Itoa(s.config.Port)
	listen, err := net.Listen("tcp4", addr)
	if err != nil {
		return fmt.Errorf("failed to start RTMP server: %v", err)
	}
	defer listen.Close()

	go func() {
		<-ctx.Done()
		listen.Close()
	}()

	for {
		conn, err := listen.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		sess := s.newMediaSession(conn)
		sess.init()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			sess.start(ctx)
		}()

		select {
		case <-ctx.Done():
			return nil
		default:
		}
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
		ctx:       s.ctx,
	}
}

func (s *MediaServer) Stop() {
	s.cancel()
	s.wg.Wait()
}
