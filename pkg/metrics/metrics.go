package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const namespace = "restreamer"

const (
	labelStream = "stream"
	labelTarget = "target"
	labelKind   = "kind"
	labelCodec  = "codec"
	labelCode   = "code"
)

var registry = prometheus.NewRegistry()

var factory = promauto.With(registry)

func init() {
	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
}

// Handler serves all registered metrics in the Prometheus exposition format.
func Handler() http.Handler {
	return promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry})
}

var (
	// IngestActive shows whether the studio is broadcasting at all
	// Alert: the publisher disconnected during a broadcast
	IngestActive = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "ingest_active",
		Help:      "1 if a publisher is currently sending the stream, 0 otherwise.",
	}, []string{labelStream})

	// IngestFrames gives the frame rate of the incoming stream per audio/video track
	// Alert: video FPS dropped or the audio track disappeared
	IngestFrames = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ingest_frames_total",
		Help:      "Frames received from the publisher.",
	}, []string{labelStream, labelKind, labelCodec})

	// IngestBytes gives the bitrate of the incoming stream
	// Alert: the publisher is connected but sends no data (OBS froze), or the
	// bitrate fell well below normal
	IngestBytes = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ingest_bytes_total",
		Help:      "Payload bytes received from the publisher.",
	}, []string{labelStream, labelKind, labelCodec})

	// IngestKeyframes gives the keyframe interval set in the studio encoder
	// Platforms expect a keyframe every 2 s and degrade or reject the stream otherwise
	// Alert: the interval is above 4 s (OBS is misconfigured)
	IngestKeyframes = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "ingest_keyframes_total",
		Help:      "Video keyframes (IDR) received from the publisher.",
	}, []string{labelStream})

	// StreamDroppedBatches counts frames lost inside the restreamer itself:
	// it could not hand them out to the platforms fast enough
	// Alert: any drop, the server is overloaded and every platform got a glitch
	StreamDroppedBatches = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "stream_dropped_batches_total",
		Help:      "Frame batches dropped because the stream dispatch buffer was full.",
	}, []string{labelStream})

	// PushConnected shows whether each platform is receiving the stream
	// Alert: a platform is not connected while the stream is live
	PushConnected = factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "push_connected",
		Help:      "1 if the push target accepted the publish, 0 otherwise.",
	}, []string{labelStream, labelTarget})

	// PushConnectFailures shows how stable the connection to a platform is
	// Short reconnects can happen between two scrapes and be missed by PushConnected
	// Alert: several reconnects within a few minutes (unstable network to the platform)
	PushConnectFailures = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "push_connect_failures_total",
		Help:      "Push target sessions that ended with an error (each one is followed by a reconnect).",
	}, []string{labelStream, labelTarget})

	// PushErrors tells why a platform rejects the stream
	// Alert: any error, with the code in the message so it is clear what to fix
	PushErrors = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "push_errors_total",
		Help:      "RTMP error statuses reported by the push target, by status code.",
	}, []string{labelStream, labelTarget, labelCode})

	// PushDroppedBatches counts frames lost on the way to one platform: the network
	// to it is slower than the stream. Viewers of that platform saw a glitch
	// Alert: any drop
	PushDroppedBatches = factory.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "push_dropped_batches_total",
		Help:      "Frame batches dropped because the queue of a connected push target overflowed.",
	}, []string{labelStream, labelTarget})
)

var pushVecs = []*prometheus.MetricVec{
	PushConnected.MetricVec,
	PushConnectFailures.MetricVec,
	PushErrors.MetricVec,
	PushDroppedBatches.MetricVec,
}

var streamVecs = append([]*prometheus.MetricVec{
	IngestActive.MetricVec,
	IngestFrames.MetricVec,
	IngestBytes.MetricVec,
	IngestKeyframes.MetricVec,
	StreamDroppedBatches.MetricVec,
}, pushVecs...)

// PushLabels returns the label set identifying a push target series.
func PushLabels(stream, target string) prometheus.Labels {
	return prometheus.Labels{labelStream: stream, labelTarget: target}
}

// DeletePushTarget removes all series of a push target, so that a removed
// target does not stay in the exported metrics forever.
func DeletePushTarget(stream, target string) {
	labels := PushLabels(stream, target)
	for _, v := range pushVecs {
		v.DeletePartialMatch(labels)
	}
}

// DeleteStream removes all series of a deleted stream.
func DeleteStream(stream string) {
	labels := prometheus.Labels{labelStream: stream}
	for _, v := range streamVecs {
		v.DeletePartialMatch(labels)
	}
}
