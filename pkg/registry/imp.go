package registry

import (
	"errors"
	"sync"
	"time"
)

type registryImpl struct {
	keys map[string]*Stream
	mux  sync.Mutex
}

func (r *registryImpl) GetStreams() ([]*Stream, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	streams := make([]*Stream, 0, len(r.keys))
	for _, stream := range r.keys {
		streams = append(streams, stream)
	}
	return streams, nil
}

func (r *registryImpl) GetStream(keyName string) (*Stream, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if key, ok := r.keys[keyName]; ok {
		return key, nil
	}
	return nil, nil
}

func (r *registryImpl) Update(key *Stream) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.keys[key.Name] = key
	return nil
}

func (r *registryImpl) DeleteStream(keyName string) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.keys, keyName)
	return nil
}

func (r *registryImpl) GetStatus(keyName string) (*StreamStatus, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		if key.status == nil {
			return &StreamStatus{
				IsLive:        false,
				LastFrameTime: 0,
				Bitrate:       0,
			}, nil
		}
		return &StreamStatus{
			IsLive:        time.Since(key.status.lastFrameTime) < 3*time.Second,
			LastFrameTime: key.status.lastFrameTime.Unix(),
			Bitrate:       key.status.bitrate,
		}, nil
	}
	return nil, errors.New("stream not found")
}

func (r *registryImpl) UpdateStatus(keyName string, lastFrameTime time.Time, bitrate uint) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		key.status = &streamStatus{
			lastFrameTime: lastFrameTime,
			bitrate:       bitrate,
		}
		return nil
	}
	return errors.New("stream not found")
}

func NewRegistry() Registry {
	return &registryImpl{
		keys: make(map[string]*Stream),
	}
}
