package registry

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

const (
	REGESTRY_STORAGE_FILE = "simple-rtmp-restreamer.data.json"
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
	r.keys[key.Name] = key
	r.mux.Unlock()
	r.savePersistent()
	return nil
}

func (r *registryImpl) DeleteStream(keyName string) error {
	r.mux.Lock()
	delete(r.keys, keyName)
	r.mux.Unlock()
	r.savePersistent()
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
	return nil, StreamNotFound{}
}

func (r *registryImpl) AddStreamTarget(keyName string, target *PushTargetUrl) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		targets := make([]*PushTargetUrl, len(key.Targets))
		copy(targets, key.Targets)
		for _, t := range targets {
			if t == target {
				return nil
			}
		}
		targets = append(targets, target)
		key.Targets = targets
		r.savePersistent()
		return nil
	}
	return StreamNotFound{}
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
	return StreamNotFound{}
}

func (r *registryImpl) loadPersistent() {
	file, err := os.Open(REGESTRY_STORAGE_FILE)
	if err != nil {
		log.Printf("Failed to load restreamser registry from file: %v", err)
		return
	}
	defer file.Close()

	var streams []Stream
	err = json.NewDecoder(file).Decode(&streams)
	if err != nil {
		log.Printf("Failed to load restreamser registry from file: %v", err)
		return
	}
	for _, stream := range streams {
		r.keys[stream.Name] = &stream
	}
}

func (r *registryImpl) savePersistent() {
	file, err := os.Create(REGESTRY_STORAGE_FILE)
	if err != nil {
		log.Printf("Failed to save restreamser registry file: %v", err)
		return
	}
	defer file.Close()

	streams, err := r.GetStreams()
	if err != nil {
		log.Printf("Failed to save restreamser registry file: %v", err)
		return
	}
	err = json.NewEncoder(file).Encode(streams)
	if err != nil {
		log.Printf("Failed to save restreamser registry file: %v", err)
		return
	}
}

func NewRegistry() Registry {
	r := registryImpl{
		keys: make(map[string]*Stream),
	}
	r.loadPersistent()
	return &r
}
