package apiserver

import (
	"errors"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"net/url"
)

type Stream struct {
	Name    string   `json:"name"`
	Targets []string `json:"targets"`
}

func (stream *Stream) toRegistryObject() (*registry.Stream, error) {
	targets := make([]*registry.PushTargetUrl, len(stream.Targets))
	for i, target := range stream.Targets {
		u, err := url.Parse(target)
		if err != nil {
			return nil, errors.New("invalid target url")
		}
		targets[i] = (*registry.PushTargetUrl)(u)
	}
	return &registry.Stream{Name: stream.Name, Targets: targets}, nil
}

func streamFromRegistryObject(stream *registry.Stream) *Stream {
	targets := make([]string, len(stream.Targets))
	for i, target := range stream.Targets {
		targets[i] = ((*url.URL)(target)).String()
	}
	return &Stream{Name: stream.Name, Targets: targets}
}
