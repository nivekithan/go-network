package protocol

import (
	"io"
	"log"
	"net"
	"sync"
	"time"
)

type LineReversalConnection struct {
	lis                 *LineReversalListener
	sessionChan         chan ClientMsg
	sessionOutgoingAddr net.Addr
	sessionToken        int
	isClosed            bool
	// TODO: Make it a fixed size byte to prevent unbounded memory usage ?
	bufferData          []byte
	bufferDataMutex     sync.Mutex
	bufferDataAvaliable sync.Cond
}

func (l *LineReversalListener) NewLineReversalConnection(newConnMsg NewConnctionMsg) *LineReversalConnection {
	conn := &LineReversalConnection{
		lis:                 l,
		sessionChan:         newConnMsg.clientMsgChan,
		sessionOutgoingAddr: newConnMsg.outgoingAddr,
		bufferData:          []byte{},
		sessionToken:        newConnMsg.sessionToken,
	}

	conn.bufferDataAvaliable = *sync.NewCond(&conn.bufferDataMutex)
	go conn.handleClientMessage()

	return conn
}

func (l *LineReversalConnection) Read(b []byte) (int, error) {

	l.bufferDataMutex.Lock()
	defer l.bufferDataMutex.Unlock()

	for len(l.bufferData) == 0 && !l.isClosed {
		l.bufferDataAvaliable.Wait()
	}

	if len(l.bufferData) == 0 {
		return 0, io.EOF

	}

	n := copy(b, l.bufferData)
	l.bufferData = l.bufferData[n:]

	return n, nil
}

func (l *LineReversalConnection) Close() {
	l.bufferDataMutex.Lock()
	defer l.bufferDataMutex.Unlock()

	l.lis.closeSession(l.sessionToken)
	l.isClosed = true
	l.bufferDataAvaliable.Broadcast()
}

// blocks the goroutine
func (conn *LineReversalConnection) handleClientMessage() {
	currentLength := 0
	timer := time.NewTimer(10 * time.Minute)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			log.Printf("Closing session due to timeout sessionToken:%d", conn.sessionToken)
			conn.Close()
		case clientMsg, ok := <-conn.sessionChan:
			if !ok {
				return
			}

			timer.Reset(10 * time.Minute)
			switch msg := clientMsg.(type) {
			case *ConnectMsg:
				ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: 0}

				conn.writeToRemote(ackMsg.toByte())

				log.Printf("sent connect ack msg %+v", ackMsg)
				continue

			case *DataMsg:
				if currentLength < msg.pos {
					// We have missed a previous msg
					ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}

					conn.writeToRemote(ackMsg.toByte())

					log.Printf("sent data ack msg %+v", ackMsg)
					continue
				}

				newData := msg.data[currentLength-msg.pos:]

				currentLength += len(newData)

				ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}

				conn.writeToRemote(ackMsg.toByte())

				log.Printf("sent data ack msg %+v", ackMsg)

				conn.writeToBuffer([]byte(newData))
				continue
			case *CloseMsg:
				closeMsg := CloseMsg{sessionToken: msg.SessionToken()}

				conn.writeToRemote(closeMsg.toByte())

				log.Printf("sent close msg %+v", closeMsg)
				continue
			}
		}
	}

}

func (conn *LineReversalConnection) writeToRemote(data []byte) {
	conn.lis.writeToRemote(data, conn.sessionOutgoingAddr)
}

func (conn *LineReversalConnection) writeToBuffer(data []byte) {
	conn.bufferDataMutex.Lock()
	defer conn.bufferDataMutex.Unlock()

	conn.bufferData = append(conn.bufferData, data...)

	conn.bufferDataAvaliable.Signal()
}
