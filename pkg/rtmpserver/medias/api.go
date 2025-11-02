package medias

import (
	"github.com/yapingcat/gomedia/go-codec"
	"time"
)

type MediaFrame struct {
	Frame    []byte
	Time     int64
	Pts      uint32
	Dts      uint32
	Cid      codec.CodecID
	IsIFrame bool
	_        [8]byte
}

func (f *MediaFrame) clone() MediaFrame {
	frames := make([]byte, len(f.Frame))
	copy(frames, f.Frame)
	return MediaFrame{
		Cid:      f.Cid,
		Pts:      f.Pts,
		Dts:      f.Dts,
		Time:     f.Time,
		Frame:    frames,
		IsIFrame: f.IsIFrame,
	}
}

type MediaFrameBatch struct {
	Frames    []MediaFrame
	StartTime time.Time
}

func (b *MediaFrameBatch) Clone() *MediaFrameBatch {
	frames := make([]MediaFrame, len(b.Frames))
	for i, frame := range b.Frames {
		frames[i] = frame.clone()
	}
	return &MediaFrameBatch{
		Frames:    frames,
		StartTime: b.StartTime,
	}
}
