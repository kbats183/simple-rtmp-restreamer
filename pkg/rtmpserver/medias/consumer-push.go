package medias

import (
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/kbats183/simple-rtmp-restreamer/pkg/api"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/metrics"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/utils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type PushConsumer struct {
	id     string
	client *rtmp.RtmpClient
	url    *url.URL
	name   string

	connMtx sync.Mutex
	conn    net.Conn

	isReady atomic.Bool
	onReady chan struct{}

	quit   chan struct{}
	quited atomic.Bool
	die    sync.Once

	framesBatches []*MediaFrameBatch
	framesMtx     sync.Mutex
	frameCome     chan struct{}

	sourceName string

	metrics pushMetrics
}

type pushMetrics struct {
	connected       prometheus.Gauge
	connectFailures prometheus.Counter
	errors          *prometheus.CounterVec
	dropped         prometheus.Counter
}

func newPushMetrics(stream, target string) pushMetrics {
	labels := metrics.PushLabels(stream, target)
	return pushMetrics{
		connected:       metrics.PushConnected.With(labels),
		connectFailures: metrics.PushConnectFailures.With(labels),
		errors:          metrics.PushErrors.MustCurryWith(labels),
		dropped:         metrics.PushDroppedBatches.With(labels),
	}
}

// NewPushConsumer creates a consumer that publishes the stream to rtmpURL.
// targetName is used in metrics and logs; if empty, the target host is used.
func NewPushConsumer(rtmpURL *api.PushTargetURL, targetName string, sourceName string) (*PushConsumer, error) {
	u := (*url.URL)(rtmpURL)
	if targetName == "" {
		targetName = u.Host
	}
	consumer := PushConsumer{
		id:            utils.GenID(),
		url:           u,
		name:          targetName,
		frameCome:     make(chan struct{}, 1),
		onReady:       make(chan struct{}),
		quit:          make(chan struct{}),
		framesBatches: make([]*MediaFrameBatch, 0, 90), // 24 + 8 * 90 = 744 bytes (64 bit)
		sourceName:    sourceName,
		metrics:       newPushMetrics(sourceName, targetName),
	}

	go func() {
		for !consumer.connect() {
			if consumer.quited.Load() {
				break
			}
		}
		log.Printf("RTMPPushClient (%s) from %s exited", consumer.url, consumer.sourceName)
	}()

	return &consumer, nil
}

func (cn *PushConsumer) connect() bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("RTMPPushClient (%s) connection panic: %v", cn.url, r)
		}
	}()
	err := cn.connection()
	if cn.quited.Load() {
		return true
	}
	cn.metrics.connectFailures.Inc()
	log.Printf("RTMPPushClient (%s) from %s failed: %v", cn.url, cn.sourceName, err)
	time.Sleep(2 * time.Second)
	return false
}

func (cn *PushConsumer) connection() error {
	host := cn.url.Host
	if cn.url.Port() == "" {
		if strings.HasPrefix(cn.url.Scheme, "rtmps") {
			host += ":443"
		} else {
			host += ":1935"
		}
	}
	var err error
	var c net.Conn
	if strings.HasPrefix(cn.url.Scheme, "rtmps") {
		conf := &tls.Config{
			ServerName:         cn.url.Hostname(),
			InsecureSkipVerify: true,
		}
		c, err = tls.Dial("tcp", host, conf)
	} else {
		c, err = net.Dial("tcp4", host)
	}
	if err != nil {
		return err
	}
	if !cn.setConn(c) {
		// Closed while dialing: do not start a session nobody will stop.
		_ = c.Close()
		return net.ErrClosed
	}

	cn.client = rtmp.NewRtmpClient(rtmp.WithComplexHandshake(), rtmp.WithEnablePublish())

	cn.client.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PUBLISH_START {
			log.Printf("RTMPPushClient (%s) ready to publish", cn.url)
			cn.isReady.Store(true)
			cn.metrics.connected.Set(1)
			cn.framesMtx.Lock()
			cn.framesBatches = nil
			cn.framesMtx.Unlock()
			cn.onReady <- struct{}{}
		}
	})
	cn.client.OnError(func(code, describe string) {
		cn.metrics.errors.WithLabelValues(code).Inc()
		log.Printf("RTMPPushClient (%s) client error: code=%s, description=%s", cn.url, code, describe)
	})
	cn.client.SetOutput(func(data []byte) error {
		_, err := c.Write(data)
		return err
	})

	// If the session ends before the target accepts the publish, onReady never
	// fires; without connDone this goroutine would leak on every reconnect.
	connDone := make(chan struct{})
	go func() {
		select {
		case <-cn.onReady:
		case <-connDone:
			return
		}
		cn.sendToServer()
		log.Printf("Stopped (%s) ...", cn.url)
	}()

	readErr := cn.socketRead(c)
	close(connDone)
	return readErr
}

func (cn *PushConsumer) Play(frame *MediaFrameBatch) {
	// if !cn.isReady.Load() {
	// 	return
	// }
	cn.framesMtx.Lock()
	if len(cn.framesBatches) >= 90 {
		// While the target is not connected the queue is discarded on connect
		// anyway, so overflow then is not a loss for viewers.
		if cn.isReady.Load() {
			cn.metrics.dropped.Add(float64(len(cn.framesBatches) - 45))
		}
		cn.framesBatches = cn.framesBatches[:45]
	}
	cn.framesBatches = append(cn.framesBatches, frame)
	cn.framesMtx.Unlock()
	select {
	case cn.frameCome <- struct{}{}:
	default:
	}
}

func (cn *PushConsumer) ID() string {
	return cn.id
}

func (cn *PushConsumer) Target() string {
	return cn.url.String()
}

func (cn *PushConsumer) TargetName() string {
	return cn.name
}

// setConn stores the connection of a new session. It returns false if the
// consumer has already been closed.
func (cn *PushConsumer) setConn(c net.Conn) bool {
	cn.connMtx.Lock()
	defer cn.connMtx.Unlock()
	if cn.quited.Load() {
		return false
	}
	cn.conn = c
	return true
}

func (cn *PushConsumer) Close() error {
	cn.quited.Store(true)
	cn.metrics.connected.Set(0)
	var err error
	cn.die.Do(func() {
		close(cn.quit)
		cn.connMtx.Lock()
		defer cn.connMtx.Unlock()
		if cn.conn != nil {
			err = cn.conn.Close()
		}
	})
	log.Printf("Closed RTMPPushConsumer %s", cn.url.String())
	return err
}

func (cn *PushConsumer) IsClosed() bool {
	return cn.quited.Load()
}

func (cn *PushConsumer) socketRead(c net.Conn) (err error) {
	cn.client.Start(cn.url.String())
	buf := make([]byte, 65536)
	var n int
	for {
		n, err = c.Read(buf)
		if err != nil && errors.Is(err, net.ErrClosed) {
			break
		} else if err != nil {
			log.Printf("RTMPPushClient (%s) from %s read error: %v", cn.url, cn.sourceName, err)
			break
		}
		err = cn.client.Input(buf[:n])
		if err != nil {
			log.Printf("RTMPPushClient (%s) handle error: %v", cn.url, err)
			break
		}
	}
	cn.isReady.Store(false)
	cn.metrics.connected.Set(0)
	return err
}

func (cn *PushConsumer) sendFrame(frame *MediaFrame) bool {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WARNING! RTMPPushClient (%s) from %s write frame error: %v", cn.url, cn.sourceName, r)
		}
	}()

	err := cn.client.WriteFrame(frame.Cid, frame.Frame, frame.Pts, frame.Dts)
	if err != nil {
		log.Printf("RTMPPushClient (%s) write socket error: %v", cn.url, err)
		return false
	}
	return true
}

func (cn *PushConsumer) sendToServer() {
	firstVideo := true
	for {
		select {
		case <-cn.frameCome:
			cn.framesMtx.Lock()
			batches := cn.framesBatches
			cn.framesBatches = nil
			cn.framesMtx.Unlock()

			for _, batch := range batches {
				for _, frame := range batch.Frames {
					if firstVideo { // wait for I frame
						if frame.IsIFrame {
							firstVideo = false
						} else {
							continue
						}
					}

					if !cn.sendFrame(&frame) {
						return
					}
				}
			}
		case <-cn.quit:
			return
		}
	}
}
