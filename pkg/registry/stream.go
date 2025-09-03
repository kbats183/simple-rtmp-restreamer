package registry

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/rtmpserver/medias"
	"log"
	"net/url"
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type Stream struct {
	Name        string               `json:"name"`
	Targets     []*api.PushTargetUrl `json:"targets"`
	TargetNames map[string]string    `json:"target_names"` // URL -> Name mapping
	status      *streamStatus

	targetConsumers []medias.MediaPushConsumer
	consumers       []medias.MediaConsumer

	framesBatches chan *medias.MediaFrameBatch

	mu     sync.Mutex
	quit   chan struct{}
	quited atomic.Bool
	die    sync.Once
}

func newStream(key *ExternalStream) (*Stream, error) {
	targets := make([]*api.PushTargetUrl, len(key.Targets))
	targetNames := make(map[string]string)

	for i, t := range key.Targets {
		parse, err := url.Parse(t.URL)
		if err != nil {
			return nil, err
		}
		targets[i] = (*api.PushTargetUrl)(parse)
		targetNames[t.URL] = t.Name
	}

	s := &Stream{
		Name:            key.Name,
		Targets:         targets,
		TargetNames:     targetNames,
		consumers:       make([]medias.MediaConsumer, 0, 10),
		targetConsumers: make([]medias.MediaPushConsumer, 0, 10),
		framesBatches:   make(chan *medias.MediaFrameBatch, 3000),
	}
	go s.dispatch()
	return s, nil
}

// TODO: REMOVE THIS!! ONLY DEBUG
////func Map[T, V any](ts []T, fn func(T) V) []V {
////	result := make([]V, len(ts))
////	for i, t := range ts {
////		result[i] = fn(t)
////	}
////	return result
//}

func (s *Stream) OnFrameBatch(frame *medias.MediaFrameBatch) {
	select {
	case s.framesBatches <- frame:
		log.Printf("Stream %s handle batch: %s", s.Name, frame.StartTime.Format(time.RFC822))
	default:
		log.Printf("Stream %s dropping frame batch due to full buffer", s.Name)
	}
}

func (s *Stream) OnProducerClose() {
	s.mu.Lock()
	consumers := slices.Clone(s.consumers)
	targetConsumers := slices.Clone(s.targetConsumers)
	s.consumers = nil
	s.targetConsumers = nil
	s.mu.Unlock()
	for _, c := range consumers {
		_ = c.Close()
	}
	for _, c := range targetConsumers {
		_ = c.Close()
	}
}

func (s *Stream) dispatch() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Stream dispatch panic for %s: %v", s.Name, r)
		}
		s.OnProducerClose()
	}()
	
	for {
		timer := time.After(time.Second * 30)
		select {
		case batch := <-s.framesBatches:
			s.updateConsumers()

			s.mu.Lock()
			targetConsumers := slices.Clone(s.targetConsumers)
			consumers := slices.Clone(s.consumers)
			s.mu.Unlock()

			// Send to consumers in parallel to avoid blocking
			var wg sync.WaitGroup
			for _, c := range targetConsumers {
				wg.Add(1)
				go func(consumer medias.MediaPushConsumer) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Panic in target consumer %s: %v", consumer.Target(), r)
						}
					}()
					consumer.Play(batch.Clone())
				}(c)
			}
			for _, c := range consumers {
				wg.Add(1)
				go func(consumer medias.MediaConsumer) {
					defer wg.Done()
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Panic in consumer %s: %v", consumer.Id(), r)
						}
					}()
					consumer.Play(batch.Clone())
				}(c)
			}
			wg.Wait()
		case <-timer:
			log.Printf("Kill stream %s after timeout", s.Name)
			s.OnProducerClose()
		case <-s.quit:
			return
		}
	}
}

func (s *Stream) updateConsumers() {
	registryTargets := make(map[string]struct{})
	for _, target := range s.Targets {
		registryTargets[target.String()] = struct{}{}
	}
	actualTargets := make(map[string]struct{})
	newTargetConsumers := make([]medias.MediaPushConsumer, 0, 10)
	for _, consumer := range s.targetConsumers {
		actualTargets[consumer.Target()] = struct{}{}
		if _, ok := registryTargets[consumer.Target()]; !ok || consumer.IsClosed() {
			go func(c medias.MediaPushConsumer) {
				if err := c.Close(); err != nil {
					log.Printf("Error closing push consumer for %s: %v", consumer.Target(), err)
				}
			}(consumer)
		} else {
			newTargetConsumers = append(newTargetConsumers, consumer)
		}
	}

	newConsumers := make([]medias.MediaConsumer, 0, 10)
	for _, consumer := range s.consumers {
		if consumer.IsClosed() {
			_ = consumer.Close()
		} else {
			newConsumers = append(newConsumers, consumer)
		}
	}

	s.mu.Lock()
	s.targetConsumers = newTargetConsumers
	s.consumers = newConsumers
	s.mu.Unlock()

	for _, target := range s.Targets {
		if _, ok := actualTargets[target.String()]; !ok {
			log.Printf("Creating PushConsumer for %s with target %s", s.Name, target.String())
			c, err := medias.NewPushConsumer(target, s.Name)
			if err != nil {
				log.Printf("Failed to create push consumer for stream %s: %v", s.Name, err)
				continue
			}
			s.addTargetConsumer(c)
		}
	}
}

// TODO: mutex security
func (s *Stream) addTargetConsumer(consumer medias.MediaPushConsumer) {
	s.mu.Lock()
	s.targetConsumers = append(s.targetConsumers, consumer)
	s.mu.Unlock()
}

func (s *Stream) AddConsumer(consumer medias.MediaConsumer) {
	s.mu.Lock()
	s.consumers = append(s.consumers, consumer)
	s.mu.Unlock()
}

func (s *Stream) RemoveConsumer(id interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, consume := range s.consumers {
		if consume.Id() == id {
			s.consumers = append(s.consumers[:i], s.consumers[i+1:]...)
		}
	}
}

func (s *Stream) Quit() {
	s.die.Do(func() {
		close(s.quit)
	})
}

type streamStatus struct {
	bitrate       uint
	lastFrameTime time.Time
}
