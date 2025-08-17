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
	"github.com/kbats183/simple-rtmp-restreamer/pkg/utils"
	"github.com/yapingcat/gomedia/go-rtmp"
)

type PushConsumer struct {
	id     string
	client *rtmp.RtmpClient
	conn   net.Conn
	url    *url.URL

	isReady atomic.Bool
	onReady chan struct{}

	quit   chan struct{}
	quited atomic.Bool
	die    sync.Once

	framesBatches []*MediaFrameBatch
	framesMtx     sync.Mutex
	frameCome     chan struct{}

	sourceName string
}

func NewPushConsumer(rtmpUrl *api.PushTargetUrl, sourceName string) (*PushConsumer, error) {
	consumer := PushConsumer{
		id:            utils.GenId(),
		url:           (*url.URL)(rtmpUrl),
		frameCome:     make(chan struct{}, 1),
		onReady:       make(chan struct{}),
		quit:          make(chan struct{}),
		framesBatches: make([]*MediaFrameBatch, 0, 90), // 24 + 8 * 90 = 744 bytes (64 bit)
		sourceName:    sourceName,
	}

	go func() {
		for {
			if consumer.connect() {
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
			InsecureSkipVerify: true,
		}
		c, err = tls.Dial("tcp", host, conf)
	} else {
		c, err = net.Dial("tcp4", host)
	}
	if err != nil {
		return err
	}

	cn.conn = c

	cn.client = rtmp.NewRtmpClient(rtmp.WithComplexHandshake(), rtmp.WithEnablePublish())

	cn.client.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PUBLISH_START {
			log.Printf("RTMPPushClient (%s) ready to publish", cn.url)
			cn.isReady.Store(true)
			cn.framesMtx.Lock()
			cn.framesBatches = nil
			cn.framesMtx.Unlock()
			cn.onReady <- struct{}{}
		}
	})
	cn.client.OnError(func(code, describe string) {
		log.Printf("RTMPPushClient (%s) client error: %s", cn.url, describe)
	})
	cn.client.SetOutput(func(data []byte) error {
		_, err := c.Write(data)
		return err
	})

	go func() {
		<-cn.onReady
		cn.sendToServer()
		log.Printf("Stopped (%s) ...", cn.url)
	}()

	return cn.socketRead()
}

func (cn *PushConsumer) Play(frame *MediaFrameBatch) {
	//if !cn.isReady.Load() {
	//	return
	//}
	cn.framesMtx.Lock()
	if len(cn.framesBatches) >= 90 {
		cn.framesBatches = cn.framesBatches[:45]
	}
	cn.framesBatches = append(cn.framesBatches, frame)
	cn.framesMtx.Unlock()
	select {
	case cn.frameCome <- struct{}{}:
	default:
	}
}

func (cn *PushConsumer) Id() string {
	return cn.id
}

func (cn *PushConsumer) Target() string {
	return cn.url.String()
}

func (cn *PushConsumer) Close() error {
	cn.quited.Store(true)
	var err error
	cn.die.Do(func() {
		close(cn.quit)
		err = cn.conn.Close()
	})
	log.Printf("Closed RTMPPushConsumer %s", cn.url.String())
	return err
}

func (cn *PushConsumer) IsClosed() bool {
	return cn.quited.Load()
}

func (cn *PushConsumer) socketRead() (err error) {
	cn.client.Start(cn.url.String())
	buf := make([]byte, 65536)
	n := 0
	for {
		n, err = cn.conn.Read(buf)
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
					if firstVideo { //wait for I frame
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
