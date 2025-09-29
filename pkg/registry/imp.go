package registry

import (
	"encoding/json"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"log"
	"net/url"
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

func (r *registryImpl) GetStreams() ([]*ExternalStream, error) {
	streams := make([]*ExternalStream, 0, len(r.keys))
	for _, key := range r.keys {
		streams = append(streams, key.toExternalStream())
	}
	return streams, nil
}

func (r *registryImpl) GetStream(keyName string) (*ExternalStream, error) {
	stream, err := r.GetInternalStream(keyName)
	if err != nil {
		return nil, err
	}
	return stream.toExternalStream(), nil
}

func (r *registryImpl) GetInternalStream(keyName string) (*Stream, error) {
	r.mux.Lock()
	defer r.mux.Unlock()
	if key, ok := r.keys[keyName]; ok {
		return key, nil
	}
	return nil, nil
}

func (r *registryImpl) Update(key *ExternalStream) error {
	err := r.updateFromExternal(key)
	if err != nil {
		return err
	}
	r.savePersistent()
	return nil
}

func (r *registryImpl) DeleteStream(keyName string) error {
	r.deleteStream(keyName)
	r.savePersistent()
	return nil
}

func (r *registryImpl) GetStatus(keyName string) (*StreamStatus, error) {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		if key.status == nil {
			return &StreamStatus{}, nil
		}
		return key.status.toStreamStatus(), nil
	}
	return nil, StreamNotFound{}
}

func (r *registryImpl) AddStreamTarget(keyName string, target *api.PushTargetUrl, targetName string) error {
	err := r.addStreamTarget(keyName, target, targetName)
	r.savePersistent()
	return err
}

func (r *registryImpl) DeleteStreamTarget(keyName string, target string) error {
	err := r.deleteStreamTarget(keyName, target)
	r.savePersistent()
	return err
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

func (r *registryImpl) GetStreamsStatus() ([]*ExternalStreamInfo, error) {
	streams := make([]*ExternalStreamInfo, 0, len(r.keys))
	for _, key := range r.keys {
		streams = append(streams, key.toExternalStreamInfo())
	}
	return streams, nil
}

func (r *registryImpl) getStreamsList() []*Stream {
	r.mux.Lock()
	defer r.mux.Unlock()
	streams := make([]*Stream, 0, len(r.keys))
	for _, stream := range r.keys {
		streams = append(streams, stream)
	}
	return streams
}

func (r *registryImpl) loadPersistent() {
	file, err := os.Open(REGESTRY_STORAGE_FILE)
	if err != nil {
		log.Printf("Failed to load restreamser registry from file: %v", err)
		return
	}
	defer file.Close()

	var streams []*ExternalStream
	err = json.NewDecoder(file).Decode(&streams)
	if err != nil {
		log.Printf("Failed to load restreamser registry from file: %v", err)
		return
	}
	for _, stream := range streams {
		s := stream
		regObj, err := newStream(s)
		if err != nil {
			log.Printf("Failed to create restreamser registry from file: %v", err)
			return
		}
		r.keys[stream.Name] = regObj
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

func (r *registryImpl) update(key *Stream) {
	r.mux.Lock()
	defer r.mux.Unlock()
	r.keys[key.Name] = key
}

func (r *registryImpl) updateFromExternal(key *ExternalStream) error {
	r.mux.Lock()
	defer r.mux.Unlock()
	if stream, ok := r.keys[key.Name]; ok {
		targets := make([]*api.PushTargetUrl, len(key.Targets))
		targetNames := make(map[string]string)

		for i, t := range key.Targets {
			parse, err := url.Parse(t.URL)
			if err != nil {
				return err
			}
			targets[i] = (*api.PushTargetUrl)(parse)
			targetNames[t.URL] = t.Name
		}

		stream.Targets = targets
		stream.TargetNames = targetNames
	} else {
		stream, err := newStream(key)
		if err != nil {
			return err
		}
		r.keys[key.Name] = stream
	}
	return nil
}

func (r *registryImpl) deleteStream(keyName string) {
	if key, ok := r.keys[keyName]; ok {
		key.Quit()
	} else {
		return
	}
	r.mux.Lock()
	defer r.mux.Unlock()
	delete(r.keys, keyName)
}

func (r *registryImpl) addStreamTarget(keyName string, target *api.PushTargetUrl, targetName string) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		targetURL := target.String()

		// Check if target already exists
		for _, t := range key.Targets {
			if t.String() == targetURL {
				return nil
			}
		}

		// Add target
		key.Targets = append(key.Targets, target)

		// Initialize TargetNames map if nil
		if key.TargetNames == nil {
			key.TargetNames = make(map[string]string)
		}

		// Store the target name
		key.TargetNames[targetURL] = targetName

		return nil
	}
	return StreamNotFound{}
}

func (r *registryImpl) deleteStreamTarget(keyName string, target string) error {
	r.mux.Lock()
	defer r.mux.Unlock()

	if key, ok := r.keys[keyName]; ok {
		newTargets := make([]*api.PushTargetUrl, 0, len(key.Targets))
		for _, t := range key.Targets {
			if t.String() != target {
				newTargets = append(newTargets, t)
			}
		}
		key.Targets = newTargets

		// Also remove from TargetNames map
		if key.TargetNames != nil {
			delete(key.TargetNames, target)
		}

		return nil
	}
	return StreamNotFound{}
}

func NewRegistry() Registry {
	r := registryImpl{
		keys: make(map[string]*Stream),
	}
	r.loadPersistent()
	return &r
}
