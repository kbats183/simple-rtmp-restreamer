package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/rtmpserver/medias"
	"github.com/yapingcat/gomedia/go-codec"
	"log"
	"sync"
	"sync/atomic"
)

type PullConsumer struct {
	sess   *MediaSession
	source *registry.Stream

	framesBatches []*medias.MediaFrameBatch
	framesMtx     sync.Mutex
	frameCome     chan struct{}

	quit       chan struct{}
	quited     atomic.Bool
	die        sync.Once
	sourceName string
}

func NewPullConsumer(sess *MediaSession, sourceName string) *PullConsumer {
	return &PullConsumer{
		sess:       sess,
		frameCome:  make(chan struct{}, 1),
		quit:       make(chan struct{}),
		sourceName: sourceName,
	}
}

func (c *PullConsumer) Play(batch *medias.MediaFrameBatch) {
	c.framesMtx.Lock()
	c.framesBatches = append(c.framesBatches, batch)
	c.framesMtx.Unlock()
	log.Printf("Send")
	select {
	case c.frameCome <- struct{}{}: //TODO: consumer common part
	default:
	}
}

func (c *PullConsumer) Id() string {
	return c.sess.id
}

func (c *PullConsumer) Close() error {
	c.quited.Store(true)
	c.sess.Close()
	c.die.Do(func() {
		close(c.quit)
	})
	log.Printf("Close PullConumer (%s) with id %s", c.sourceName, c.Id())
	return nil
}

func (c *PullConsumer) IsClosed() bool {
	return c.quited.Load()
}

func (c *PullConsumer) sendToClient() {
	log.Printf("PullConsumer dispatch (%s) id %s", c.sourceName, c.Id())
	firstVideo := true
	for {
		select {
		case <-c.frameCome:
			c.framesMtx.Lock()
			batches := c.framesBatches
			c.framesBatches = nil
			c.framesMtx.Unlock()

			log.Printf("Process %d frame bathces", len(batches))
			for _, batch := range batches {
				//bytes := 0
				for _, frame := range batch.Frames {
					if firstVideo { //wait for I frame
						if frame.Cid == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(frame.Frame) {
							firstVideo = false
						} else if frame.Cid == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(frame.Frame) {
							firstVideo = false
						} else {
							continue
						}
					}

					defer func() {
						if r := recover(); r != nil {
							log.Printf("WARNING! RTMPPullClient (%s) from %s write frame error: %v", c.Id(), c.sourceName, r)
						}
					}()

					err := c.sess.handle.WriteFrame(frame.Cid, frame.Frame, frame.Pts, frame.Dts)
					if err != nil {
						log.Printf("RTMPPullClient (%s) write socket error: %v", c.Id(), err)
						return
					}
					//bytes += len(frame.frame)
				}

				//log.Printf("Push %d kbps", bytes*8/1024)
			}
		case <-c.quit:
			return
		}
	}
}
