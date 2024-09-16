package rtmpserver

import (
	"github.com/yapingcat/gomedia/go-codec"
	"time"
)

type MediaServerConfig struct {
	Port int
}

type MediaFrame struct {
	time  time.Time
	cid   codec.CodecID
	frame []byte
	pts   uint32
	dts   uint32
}

func (f *MediaFrame) clone() MediaFrame {
	frames := make([]byte, len(f.frame))
	copy(frames, f.frame)
	return MediaFrame{
		cid:   f.cid,
		pts:   f.pts,
		dts:   f.dts,
		time:  f.time,
		frame: frames,
	}
}

type MediaFrameBatch struct {
	frames    []MediaFrame
	startTime time.Time
}

func (b *MediaFrameBatch) clone() *MediaFrameBatch {
	frames := make([]MediaFrame, len(b.frames))
	for i, frame := range b.frames {
		frames[i] = frame.clone()
	}
	return &MediaFrameBatch{
		frames:    frames,
		startTime: b.startTime,
	}
}
