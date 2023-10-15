package udp

import (
	"errors"
	"fmt"
	"github.com/snsinfu/reverse-tunnel/server/service"
	"io"
	"net"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/gorilla/websocket"
	"github.com/snsinfu/reverse-tunnel/config"
)

// sessionTimeout is the timeout used to kill idle UDP session.
const sessionTimeout = 300 * time.Second

// ErrTunnelNotReady is returned when an application tries to send data through
// a session that does not have an established tunnel connection yet.
var ErrTunnelNotReady = errors.New("tunnel is not yet established")

// Session implements service.Session for UDP tunneling.
type Session struct {
	tunnel *websocket.Conn
	conn   *net.UDPConn
	peer   *net.UDPAddr
	idle   int32
	port   int
}

// NewSession creates a Session for tunneling UDP packets from/to given peer.
func NewSession(port int, conn *net.UDPConn, peer *net.UDPAddr) *Session {
	return &Session{
		port: port,
		conn: conn,
		peer: peer,
	}
}

// SendToClient sends msg to UDP client. Calling this function resets internal
// idle counter.
func (sess *Session) SendToClient(msg []byte) error {
	atomic.StoreInt32(&sess.idle, 0)

	_, err := sess.conn.WriteToUDP(msg, sess.peer)
	return err
}

// SendToAgent sends msg to the other end of the tunnel if tunnel is ready.
// Calling this function resets internal idle counter.
func (sess *Session) SendToAgent(msg []byte) error {
	atomic.StoreInt32(&sess.idle, 0)

	ws := (*websocket.Conn)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&sess.tunnel))))
	if ws == nil {
		return ErrTunnelNotReady
	}

	return ws.WriteMessage(websocket.BinaryMessage, msg)
}

// PeerAddr returns the address of the connected client.
func (sess *Session) PeerAddr() net.Addr {
	return sess.peer
}

// Start starts downlink tunneling. Uplink needs to be handled in the listener
// loop due to the connection-less nature of UDP.
func (sess *Session) Start(ws *websocket.Conn) error {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(&sess.tunnel)), unsafe.Pointer(ws))

	downlinkCounter := service.DownlinkBytesCounterVec.WithLabelValues(fmt.Sprintf("%d/udp", sess.port))

	// Kill idle session
	go func() {
		for range time.NewTicker(sessionTimeout / 2).C {
			if atomic.AddInt32(&sess.idle, 1) == 2 {
				break
			}
		}

		ws.Close()
	}()

	// Downlink
	buf := make([]byte, config.BufferSize)

	for {
		_, r, err := ws.NextReader()
		if err != nil {
			return err
		}

		for {
			n, err := r.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			if err := sess.SendToClient(buf[:n]); err != nil {
				return err
			}
			downlinkCounter.Add(float64(n))
		}
	}
}

// Close does nothing because UDP has no real connection.
func (sess Session) Close() error {
	return nil
}
