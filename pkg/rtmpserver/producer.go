package rtmpserver

import (
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/rtmpserver/medias"
	"github.com/yapingcat/gomedia/go-codec"
	"sync"
	"time"
)

//var mainProducer *MediaProducer = nil

type MediaProducer struct {
	name               string
	session            *MediaSession
	mtx                sync.Mutex
	frames             chan *medias.MediaFrame
	currentFramesBatch *medias.MediaFrameBatch
	quit               chan struct{}
	die                sync.Once
	stream             *registry.Stream
}

func newMediaProducer(name string, sess *MediaSession, stream *registry.Stream) *MediaProducer {
	return &MediaProducer{
		name:               name,
		session:            sess,
		currentFramesBatch: nil,
		quit:               make(chan struct{}),
		stream:             stream,
	}
}

func (prod *MediaProducer) start() {
	sess := prod.session
	sess.handle.OnFrame(func(cid codec.CodecID, pts, dts uint32, frame []byte) {
		if prod.currentFramesBatch == nil {
			prod.currentFramesBatch = &medias.MediaFrameBatch{StartTime: time.Now()}
		}
		isIFrame := cid == codec.CODECID_VIDEO_H264 && codec.IsH264IDRFrame(frame) ||
			cid == codec.CODECID_VIDEO_H265 && codec.IsH265IDRFrame(frame)
		prod.currentFramesBatch.Frames = append(prod.currentFramesBatch.Frames, medias.MediaFrame{
			Cid:      cid,
			Frame:    frame,
			Pts:      pts,
			Dts:      dts,
			Time:     time.Now(),
			IsIFrame: isIFrame,
		})

		since := time.Since(prod.currentFramesBatch.StartTime)
		if since >= time.Second || isIFrame {
			var bytes int
			for _, mediaFrame := range prod.currentFramesBatch.Frames {
				bytes += len(mediaFrame.Frame)
			}
			_ = sess.registry.UpdateStatus(prod.name, prod.currentFramesBatch.StartTime, evaluateBitrate(bytes, since))
			prod.stream.OnFrameBatch(prod.currentFramesBatch)
			prod.currentFramesBatch = nil
		}
	})
}

func evaluateBitrate(bytes int, since time.Duration) uint {
	if since == 0 {
		return 0
	}
	return uint(time.Duration(bytes) * time.Second / since / 128)
}

func (prod *MediaProducer) stop() {
	prod.die.Do(func() {
		close(prod.quit)
		//center.unRegister(prod.name) // should i unregister here?
	})
}

func (prod *MediaProducer) Close() error {
	//prod.stream.OnProducerClose()
	prod.stop()
	_ = prod.session.registry.UpdateStatus(prod.name, time.Unix(0, 0), 0)
	return nil
}

func (prod *MediaProducer) debugName() string {
	if prod.session == nil {
		return prod.name
	}
	return prod.name + "-" + prod.session.id
}
