package medias

type MediaConsumer interface {
	Play(batch *MediaFrameBatch)
	ID() string
	IsClosed() bool
	Close() error
}

type MediaPushConsumer interface {
	MediaConsumer
	Target() string
	// TargetName is a human-readable target identifier that is safe to expose
	// in metrics and logs (unlike Target, it never contains the stream key).
	TargetName() string
}
