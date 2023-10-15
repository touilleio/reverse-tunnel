package tcp

import (
	"errors"
	"fmt"
	"github.com/snsinfu/reverse-tunnel/server/service"
	"io"
	"log"
	"net"

	"github.com/gorilla/websocket"
	"github.com/snsinfu/go-taskch"
	"github.com/snsinfu/reverse-tunnel/config"
)

// Session implements service.Session for TCP tunneling.
type Session struct {
	conn *net.TCPConn
	port int
}

// NewSession creates a Session for tunneling given TCP connection.
func NewSession(port int, conn *net.TCPConn) *Session {
	return &Session{
		port: port,
		conn: conn,
	}
}

// PeerAddr returns the address of the connected client.
func (sess Session) PeerAddr() net.Addr {
	return sess.conn.RemoteAddr()
}

// Start starts tunneling TCP packets through given websocket channel.
func (sess Session) Start(ws *websocket.Conn) error {
	defer sess.conn.Close()

	tasks := taskch.New()

	uplinkCounter := service.UplinkBytesCounterVec.WithLabelValues(fmt.Sprintf("%d/tcp", sess.port))
	downlinkCounter := service.DownlinkBytesCounterVec.WithLabelValues(fmt.Sprintf("%d/tcp", sess.port))

	// Uplink
	tasks.Go(func() error {
		buf := make([]byte, config.BufferSize)

		for {
			n, err := sess.conn.Read(buf)
			if err != nil {
				return err
			}

			if err := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return err
			}
			uplinkCounter.Add(float64(n))
		}
	})

	// Downlink
	tasks.Go(func() error {
		buf := make([]byte, config.BufferSize)

		for {
			_, r, err := ws.NextReader()
			if err != nil {
				return err
			}

			n, err := io.CopyBuffer(sess.conn, r, buf)
			if err != nil {
				return err
			}
			downlinkCounter.Add(float64(n))
		}
	})

	err := tasks.Wait()

	if errors.Is(err, io.EOF) {
		log.Printf("Client %s closed normally. Closing session.", sess.conn.RemoteAddr())
		return nil
	}

	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		log.Printf("Session closed normally. Closing client %s.", sess.conn.RemoteAddr())
		return nil
	}

	return err
}

// Close closes client connection.
func (sess Session) Close() error {
	return sess.conn.Close()
}
