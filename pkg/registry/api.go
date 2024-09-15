package registry

import (
	"errors"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"net/url"
	"time"
)

type StreamStatus struct {
	IsLive        bool  `json:"is_live"`
	Bitrate       uint  `json:"bitrate"`
	LastFrameTime int64 `json:"last_frame_time"`
}

type Registry interface {
	GetStreams() ([]*ExternalStream, error) // should it public?
	GetStream(keyName string) (*ExternalStream, error)
	GetInternalStream(keyName string) (*Stream, error)
	Update(key *ExternalStream) error
	DeleteStream(keyName string) error
	AddStreamTarget(keyName string, target *api.PushTargetUrl) error
	GetStatus(keyName string) (*StreamStatus, error)
	UpdateStatus(keyName string, lastFrameTime time.Time, bitrate uint) error
}

type ExternalStream struct {
	Name    string   `json:"name"`
	Targets []string `json:"targets"`
}

func (stream *ExternalStream) toRegistryObject() (*Stream, error) {
	targets := make([]*api.PushTargetUrl, len(stream.Targets))
	for i, target := range stream.Targets {
		u, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("invalid target url")
		}
		targets[i] = (*api.PushTargetUrl)(u)
	}
	s, err := newStream(stream)
	return s, err
}

func (stream *Stream) toExternalStream() *ExternalStream {
	targets := make([]string, len(stream.Targets))
	for i, target := range stream.Targets {
		targets[i] = ((*url.URL)(target)).String()
	}
	return &ExternalStream{Name: stream.Name, Targets: targets}
}
