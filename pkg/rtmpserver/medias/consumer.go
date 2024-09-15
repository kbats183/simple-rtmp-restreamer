package medias

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/utils"
	"log"
	"net/url"
	"time"
)

type MediaConsumer interface {
	Play(batch *MediaFrameBatch)
	Id() string
	IsClosed() bool
	Close() error
}

type MediaPushConsumer interface {
	MediaConsumer
	Target() string
}

func NewPushConsumer(rtmpUrl *api.PushTargetUrl, sourceName string) (*PushConsumer, error) {
	consumer := PushConsumer{
		id:         utils.GenId(),
		url:        (*url.URL)(rtmpUrl),
		frameCome:  make(chan struct{}, 1),
		onReady:    make(chan struct{}),
		quit:       make(chan struct{}),
		sourceName: sourceName,
	}

	go func() {
		for {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("RTMPPushClient (%s) connection panic: %v", consumer.url, r)
				}
			}()
			err := consumer.connection()
			if consumer.quited.Load() {
				break
			}
			log.Printf("RTMPPushClient (%s) from %s failed: %v", consumer.url, consumer.sourceName, err)
			time.Sleep(2 * time.Second)
		}
		log.Printf("RTMPPushClient (%s) from %s exited", consumer.url, consumer.sourceName)
	}()

	return &consumer, nil
}
