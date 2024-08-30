package registry

import (
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
	GetStreams() ([]*Stream, error)
	GetStream(keyName string) (*Stream, error)
	Update(key *Stream) error
	DeleteStream(keyName string) error
	GetStatus(keyName string) (*StreamStatus, error)
	UpdateStatus(keyName string, lastFrameTime time.Time, bitrate uint) error
}
