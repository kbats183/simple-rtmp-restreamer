package rtmpserver

import (
	"errors"
	"fmt"
	"github.com/kbats183/simple-rtmp-restreamer/pkg/registry"
	"github.com/yapingcat/gomedia/go-rtmp"
	"io"
	"log"
	"net"
	"sync"
)

type MediaSession struct {
	id     string
	conn   net.Conn
	handle *rtmp.RtmpServerHandle

	frameCome chan struct{}
	quit      chan struct{}
	resource  io.Closer
	die       sync.Once //?
	registry  registry.Registry

	producer *MediaProducer
}

func (sess *MediaSession) init() {
	sess.handle.SetOutput(func(b []byte) error {
		_, err := sess.conn.Write(b)
		return err
	})

	sess.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		return rtmp.NETSTREAM_PLAY_NOTFOUND
		//if mainProducer == nil {
		//	return rtmp.NETSTREAM_PLAY_NOTFOUND
		//}
		//return rtmp.NETSTREAM_PLAY_START
	})

	sess.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		stream, err := sess.registry.GetStream(streamName)
		if err != nil {
			log.Printf("Failed to get %s stream info: %v", streamName, err)
			return rtmp.NETSTREAM_CONNECT_FAILED
		} else if stream == nil {
			log.Printf("No such %s stream info", streamName)
			return rtmp.NETSTREAM_CONNECT_FAILED
		}

		p := newMediaProducer(streamName, sess)
		sess.producer = p

		return rtmp.NETSTREAM_PUBLISH_START
	})

	sess.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			fmt.Println("play start")
			//name := sess.handle.GetStreamName()
			//source := center.find(name)
			//sess.source = source
			//if mainProducer != nil {
			//	cons := &PullConsumer{sess: sess}
			//	sess.resource = cons
			//	mainProducer.addConsumer(cons)
			//	fmt.Println("ready to play")
			//	go cons.sendToClient()
			//}
		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			name := sess.handle.GetStreamName()
			log.Printf("New rtmp stream %s", name)

			sess.resource = sess.producer
			sess.producer.start()
		} else {
			log.Printf("New state of %s: %d", sess.handle.GetStreamName(), newState)
		}
	})
}

func (sess *MediaSession) start() {
	defer sess.stop()
	for {
		buf := make([]byte, 65536)
		n, err := sess.conn.Read(buf)
		if err != nil && errors.Is(err, io.EOF) {
			return
		} else if err != nil {
			log.Printf("MediaSession read error: %v", err)
			return
		}
		err = sess.handle.Input(buf[:n])
		if err != nil {
			log.Printf("MediaSession handle error: %v", err)
			return
		}
	}
}

func (sess *MediaSession) stop() {
	if sess.resource != nil {
		_ = sess.resource.Close()
		sess.resource = nil
	}
	_ = sess.Close()
}

func (sess *MediaSession) Close() error {
	sess.die.Do(func() {
		close(sess.quit)
		_ = sess.conn.Close()
	})
	return nil
}
