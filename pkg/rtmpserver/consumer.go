package rtmpserver

import (
	"github.com/yapingcat/gomedia/go-codec"
	"sync"
)

type MediaConsumer interface {
	Play(batch *MediaFrameBatch)
	Id() string
	Close() error
}

type PullConsumer struct {
	sess       *MediaSession
	batches    []*MediaFrameBatch
	batchesMtx sync.Mutex
	source     *MediaProducer
}

func (c *PullConsumer) Play(batch *MediaFrameBatch) {
	c.batchesMtx.Lock()
	c.batches = append(c.batches, batch)
	c.batchesMtx.Unlock()
	select {
	case c.sess.frameCome <- struct{}{}: //TODO: consumer common part
	default:
	}
}

func (c *PullConsumer) Id() string {
	return c.sess.id
}

func (c *PullConsumer) Close() error {
	if c.source != nil {
		c.source.removeConsumer(c.Id())
		c.source = nil
	}
	c.sess.Close()
	return nil
}

func (c *PullConsumer) sendToClient() {
	defer c.Close()

	firstVideo := true
	for {
		select {
		case <-c.sess.frameCome:
			c.batchesMtx.Lock()
			batches := c.batches
			c.batches = nil
			c.batchesMtx.Unlock()
			for _, batch := range batches {
				for _, frame := range batch.frames {
					if firstVideo { //wait for I frame
						if frame.cid == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(frame.frame) {
							firstVideo = false
						} else if frame.cid == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(frame.frame) {
							firstVideo = false
						} else {
							continue
						}
					}
					err := c.sess.handle.WriteFrame(frame.cid, frame.frame, frame.pts, frame.dts)
					if err != nil {
						_ = c.Close()
						return
					}
				}
			}
		case <-c.sess.quit:
			return
		}
	}
}
