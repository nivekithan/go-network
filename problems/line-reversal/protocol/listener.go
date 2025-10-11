package protocol

import (
	"errors"
	"log"
	"net"
	"sync"
)

type NewConnctionMsg struct {
	clientMsgChan chan ClientMsg
	outgoingAddr  net.Addr
}

type LineReversalListener struct {
	conn               net.PacketConn
	sessionTokenToChan map[int]chan ClientMsg

	closeMutex  sync.Mutex
	newConnChan chan NewConnctionMsg
	isClosed    bool
	closeChan   chan struct{}
}

func NewListener(addr string) (*LineReversalListener, error) {
	conn, err := net.ListenPacket("udp", addr)

	if err != nil {
		return nil, err
	}

	listener := &LineReversalListener{
		conn:               conn,
		sessionTokenToChan: make(map[int]chan ClientMsg),
		isClosed:           false,
		newConnChan:        make(chan NewConnctionMsg),
		closeChan:          make(chan struct{}),
	}

	go listener.handlePacketConnection(conn)

	return listener, nil
}

func (l *LineReversalListener) Accept() (*LineReversalConnection, error) {

	if l.isClosed {
		return nil, net.ErrClosed
	}

	select {
	case <-l.closeChan:
		return nil, net.ErrClosed
	case clientMsgChan := <-l.newConnChan:
		conn := l.NewLineReversalConnection(clientMsgChan)

		return conn, nil
	}
}

func (l *LineReversalListener) writeToRemote(data []byte, addr net.Addr) {

	if _, err := l.conn.WriteTo(data, addr); err != nil {
		// TODO: Handle the error appropriately
		panic(err)
	}

}

func (l *LineReversalListener) Close() error {
	l.closeMutex.Lock()
	defer l.closeMutex.Unlock()

	if l.isClosed {
		return nil
	}

	l.isClosed = true

	close(l.closeChan)

	return l.conn.Close()
}

func (l *LineReversalListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

// Blocks the current goroutine
func (l *LineReversalListener) handlePacketConnection(conn net.PacketConn) {
	l.handlePacketConnectionImpl(conn)

	l.Close()

}

// Blocks the current goroutine
// Call handlePacketConnect
func (l *LineReversalListener) handlePacketConnectionImpl(conn net.PacketConn) error {

	for {
		var packet [1000]byte

		n, outgoingAddr, err := conn.ReadFrom(packet[:])

		if err != nil {
			log.Println(err)
			return errors.New("Unable to read from udp packetConn")
		}

		log.Printf("Got new packet of size: %d", n)

		clientMsg, err := ParsePacketData(string(packet[:n]))

		if err != nil {
			log.Printf("got invalid packet data: %v. Ignoring this packet\n", err)
			continue
		}

		switch msg := (clientMsg).(type) {
		case *ConnectMsg:
			sessionChan, ok := l.sessionTokenToChan[msg.SessionToken()]

			if !ok {
				sessionChan := make(chan ClientMsg)
				l.sessionTokenToChan[msg.SessionToken()] = sessionChan
				l.newConnChan <- NewConnctionMsg{clientMsgChan: sessionChan, outgoingAddr: outgoingAddr}
				sessionChan <- clientMsg
				continue
			}

			sessionChan <- clientMsg
		default:
			sessionChan, ok := l.sessionTokenToChan[msg.SessionToken()]

			if !ok {
				// TODO: Send closeSession msg instead
				log.Printf("got msgType: %v, but there is no session for :%v. Therefore ignoring the packet\n", msg.Type(), msg.SessionToken())
				continue
			}

			sessionChan <- clientMsg
		}
	}
}
