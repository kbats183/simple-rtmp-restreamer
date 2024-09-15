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

	quit     chan struct{}
	resource io.Closer
	die      sync.Once //?
	registry registry.Registry

	producer     *MediaProducer
	pullConsumer *PullConsumer
}

func (sess *MediaSession) init() {
	sess.handle.SetOutput(func(b []byte) error {
		_, err := sess.conn.Write(b)
		return err
	})

	sess.handle.OnPlay(func(app, streamName string, start, duration float64, reset bool) rtmp.StatusCode {
		stream, err := sess.registry.GetInternalStream(streamName)
		if err != nil {
			log.Printf("Failed to get InternalStreamer for pull consumer %s: %v", streamName, err)
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		if stream == nil {
			return rtmp.NETSTREAM_PLAY_NOTFOUND
		}
		sess.pullConsumer = NewPullConsumer(sess, streamName)
		stream.AddConsumer(sess.pullConsumer)
		return rtmp.NETSTREAM_PLAY_START
	})

	sess.handle.OnPublish(func(app, streamName string) rtmp.StatusCode {
		stream, err := sess.registry.GetInternalStream(streamName)
		if err != nil {
			log.Printf("Failed to get %s stream info: %v", streamName, err)
			return rtmp.NETCONNECT_CONNECT_REJECTED
		} else if stream == nil {
			log.Printf("No such %s stream info", streamName)
			return rtmp.NETCONNECT_CONNECT_REJECTED
		}

		p := newMediaProducer(streamName, sess, stream)
		sess.producer = p

		return rtmp.NETSTREAM_PUBLISH_START
	})

	sess.handle.OnStateChange(func(newState rtmp.RtmpState) {
		if newState == rtmp.STATE_RTMP_PLAY_START {
			fmt.Println("play start")

			if sess.pullConsumer == nil {
				return
			}
			go sess.pullConsumer.sendToClient()
			sess.resource = sess.pullConsumer
		} else if newState == rtmp.STATE_RTMP_PUBLISH_START {
			name := sess.handle.GetStreamName()
			log.Printf("New rtmp stream %s", name)

			sess.resource = sess.producer
			sess.producer.start()
		} else if newState == rtmp.STATE_RTMP_PUBLISH_FAILED {
			name := sess.handle.GetStreamName()
			log.Printf("Failed rtmp stream %s", name)
			sess.stop()
		} else {
			//log.Printf("New state of %s: %d", sess.handle.GetStreamName(), newState)
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
