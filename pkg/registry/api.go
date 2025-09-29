package registry

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"net/url"
	"time"
)

type Registry interface {
	GetStreams() ([]*ExternalStream, error) // should it public?
	GetStream(keyName string) (*ExternalStream, error)
	GetInternalStream(keyName string) (*Stream, error)
	Update(key *ExternalStream) error
	DeleteStream(keyName string) error
	AddStreamTarget(keyName string, target *api.PushTargetUrl, targetName string) error
	DeleteStreamTarget(keyName string, target string) error
	GetStatus(keyName string) (*StreamStatus, error)
	GetStreamsStatus() ([]*ExternalStreamInfo, error) // should it public?
	UpdateStatus(keyName string, lastFrameTime time.Time, bitrate uint) error
}

type PushTarget struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type ExternalStream struct {
	Name    string       `json:"name"`
	Targets []PushTarget `json:"targets"`
}

type StreamStatus struct {
	IsLive        bool  `json:"is_live"`
	Bitrate       uint  `json:"bitrate"`
	LastFrameTime int64 `json:"last_frame_time"`
}

type ExternalStreamInfo struct {
	ExternalStream
	Status *StreamStatus `json:"status"`
}

func (stream *ExternalStream) toRegistryObject() (*Stream, error) {
	s, err := newStream(stream)
	return s, err
}

func (stream *Stream) toExternalStream() *ExternalStream {
	targets := make([]PushTarget, len(stream.Targets))
	for i, target := range stream.Targets {
		targetURL := ((*url.URL)(target)).String()
		targetName := targetURL // Default to URL if no name stored

		if stream.TargetNames != nil {
			if name, exists := stream.TargetNames[targetURL]; exists && name != "" {
				targetName = name
			}
		}

		targets[i] = PushTarget{
			Name: targetName,
			URL:  targetURL,
		}
	}
	return &ExternalStream{Name: stream.Name, Targets: targets}
}

func (status *streamStatus) toStreamStatus() *StreamStatus {
	if status == nil {
		return &StreamStatus{}
	}
	return &StreamStatus{
		IsLive:        time.Since(status.lastFrameTime) < 3*time.Second,
		LastFrameTime: status.lastFrameTime.Unix(),
		Bitrate:       status.bitrate,
	}
}

func (stream *Stream) toExternalStreamInfo() *ExternalStreamInfo {
	es := stream.toExternalStream()
	return &ExternalStreamInfo{ExternalStream: *es, Status: stream.status.toStreamStatus()}
}
