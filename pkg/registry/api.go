package registry

import (
	"errors"
	"net/url"
	"time"
)

type PushTargetUrl url.URL

type Stream struct {
	Name    string           `json:"name"`
	Targets []*PushTargetUrl `json:"targets"`
	status  *streamStatus
}

type streamStatus struct {
	bitrate       uint
	lastFrameTime time.Time
}

type StreamStatus struct {
	IsLive        bool  `json:"is_live"`
	Bitrate       uint  `json:"bitrate"`
	LastFrameTime int64 `json:"last_frame_time"`
}

type Registry interface {
	GetStreams() ([]*ExternalStream, error) // should it public?
	GetStream(keyName string) (*ExternalStream, error)
	Update(key *ExternalStream) error
	DeleteStream(keyName string) error
	AddStreamTarget(keyName string, target *PushTargetUrl) error
	GetStatus(keyName string) (*StreamStatus, error)
	UpdateStatus(keyName string, lastFrameTime time.Time, bitrate uint) error
}

func (p *PushTargetUrl) String() string {
	return (*url.URL)(p).String()
}

type ExternalStream struct {
	Name    string   `json:"name"`
	Targets []string `json:"targets"`
}

func (stream *ExternalStream) toRegistryObject() (*Stream, error) {
	targets := make([]*PushTargetUrl, len(stream.Targets))
	for i, target := range stream.Targets {
		u, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("invalid target url")
		}
		targets[i] = (*PushTargetUrl)(u)
	}
	return &Stream{Name: stream.Name, Targets: targets}, nil
}

func (stream *Stream) toExternalStream() *ExternalStream {
	targets := make([]string, len(stream.Targets))
	for i, target := range stream.Targets {
		targets[i] = ((*url.URL)(target)).String()
	}
	return &ExternalStream{Name: stream.Name, Targets: targets}
}
