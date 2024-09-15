package rtmpserver

import (
	"crypto/tls"
	"errors"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/yapingcat/gomedia/go-codec"
	"github.com/yapingcat/gomedia/go-rtmp"
	"log"
	"net"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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

func (cn *PushConsumer) connection() error {
	host := cn.url.Host
	if cn.url.Port() == "" {
		host += ":1935"
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

func (cn *PushConsumer) sendToServer() {
	firstVideo := true
	// TODO: Maybe we need to call Close method in the defer!
	defer func() {
		if r := recover(); r != nil {
			log.Printf("WARNING! RTMPPushClient (%s) from %s write frame error: %v", cn.url, cn.sourceDebugName(), r)
		}
	}()
	for {
		select {
		case <-cn.frameCome:
			cn.framesMtx.Lock()
			batches := cn.framesBatches
			cn.framesBatches = nil
			cn.framesMtx.Unlock()

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

					err := cn.client.WriteFrame(frame.Cid, frame.Frame, frame.Pts, frame.Dts)
					if err != nil {
						log.Printf("RTMPPushClient (%s) write socket error: %v", cn.url, err)
						return
					}
				}
			}
		case <-cn.quit:
			return
		}
	}
}
