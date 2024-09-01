package rtmpserver

import (
	"github.com/yapingcat/gomedia/go-codec"
	"log"
	"net/url"
	"slices"
	"sync"
	"time"
)

//var mainProducer *MediaProducer = nil

type MediaProducer struct {
	name               string
	session            *MediaSession
	mtx                sync.Mutex
	targetConsumers    []*PushConsumer
	consumers          []MediaConsumer
	frames             chan *MediaFrame
	framesBatches      chan *MediaFrameBatch
	currentFramesBatch *MediaFrameBatch
	quit               chan struct{}
	die                sync.Once
}

func newMediaProducer(name string, sess *MediaSession) *MediaProducer {
	return &MediaProducer{
		name:               name,
		session:            sess,
		consumers:          make([]MediaConsumer, 0, 10),
		targetConsumers:    make([]*PushConsumer, 0, 10),
		framesBatches:      make(chan *MediaFrameBatch, 3000),
		currentFramesBatch: nil,
		quit:               make(chan struct{}),
	}
}

func (prod *MediaProducer) start() {
	sess := prod.session
	sess.handle.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
		if prod.currentFramesBatch == nil {
			prod.currentFramesBatch = &MediaFrameBatch{startTime: time.Now()}
		}
		prod.currentFramesBatch.frames = append(prod.currentFramesBatch.frames, MediaFrame{
			cid:   cid,
			frame: frame,
			pts:   pts,
			dts:   dts,
			time:  time.Now(),
		})

		since := time.Since(prod.currentFramesBatch.startTime)
		if since >= time.Second {
			if stream, err := sess.registry.GetStream(prod.name); err == nil {
				actualTargets := make(map[url.URL]struct{})
				for _, consumer := range prod.targetConsumers {
					actualTargets[*consumer.url] = struct{}{}
				}
				for _, target := range stream.Targets {
					targetUrl := *(*url.URL)(target)
					if _, ok := actualTargets[targetUrl]; !ok {
						log.Printf("Creating PushConsumer for %s with target %s", prod.name, targetUrl.String())
						c, err := NewPushConsumer(target)
						if err != nil {
							log.Printf("Failed to create push consumer for stream %s: %v", prod.name, err)
							continue
						}
						prod.addTargetConsumer(c)
					}
				}
			}

			var bytes int
			for _, mediaFrame := range prod.currentFramesBatch.frames {
				bytes += len(mediaFrame.frame)
			}
			_ = sess.registry.UpdateStatus(prod.name, prod.currentFramesBatch.startTime, uint(time.Duration(bytes)*time.Second/since/128))
			prod.framesBatches <- prod.currentFramesBatch
			prod.currentFramesBatch = nil
		}
	})

	go prod.dispatch()
}

func (prod *MediaProducer) stop() {
	prod.die.Do(func() {
		close(prod.quit)
		//center.unRegister(prod.name) // should i unregister here?
	})
}

func (prod *MediaProducer) Close() error {
	prod.mtx.Lock()
	consumers := slices.Clone(prod.consumers)
	targetConsumers := slices.Clone(prod.targetConsumers)
	prod.mtx.Unlock()

	for _, c := range consumers {
		_ = c.Close()
	}
	for _, c := range targetConsumers {
		_ = c.Close()
	}
	prod.stop()
	return nil
}

func (prod *MediaProducer) dispatch() {
	for {
		select {
		case batch := <-prod.framesBatches:
			log.Printf("Prudecer dispatch %d frames batch %d", len(batch.frames), batch.frames[0].dts)

			prod.mtx.Lock()
			targetConsumers := slices.Clone(prod.targetConsumers)
			consumers := slices.Clone(prod.consumers)
			prod.mtx.Unlock()

			for _, c := range targetConsumers {
				c.Play(batch.clone())
			}
			for _, c := range consumers {
				c.Play(batch.clone())
			}
		case <-prod.session.quit:
			return
		case <-prod.quit:
			return
		}
	}
}

func (prod *MediaProducer) addConsumer(consumer MediaConsumer) {
	prod.mtx.Lock()
	prod.consumers = append(prod.consumers, consumer)
	prod.mtx.Unlock()
}

func (prod *MediaProducer) addTargetConsumer(consumer *PushConsumer) {
	prod.mtx.Lock()
	prod.targetConsumers = append(prod.targetConsumers, consumer)
	prod.mtx.Unlock()
}

func (prod *MediaProducer) removeConsumer(id interface{}) {
	prod.mtx.Lock()
	defer prod.mtx.Unlock()
	for i, consume := range prod.consumers {
		if consume.Id() == id {
			prod.consumers = append(prod.consumers[i:], prod.consumers[i+1:]...)
		}
	}
}
