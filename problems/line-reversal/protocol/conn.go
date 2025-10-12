package protocol

import (
	"log"
	"net"
	"sync"
)

type LineReversalConnection struct {
	lis                 *LineReversalListener
	sessionChan         chan ClientMsg
	sessionOutgoingAddr net.Addr
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
	}

	conn.bufferDataAvaliable = *sync.NewCond(&conn.bufferDataMutex)
	go conn.handleClientMessage()

	return conn
}

func (l *LineReversalConnection) Read(b []byte) (int, error) {

	l.bufferDataMutex.Lock()
	defer l.bufferDataMutex.Unlock()

	for len(l.bufferData) == 0 {
		l.bufferDataAvaliable.Wait()
	}

	n := copy(b, l.bufferData)
	l.bufferData = l.bufferData[n:]

	return n, nil
}

// blocks the goroutine
func (conn *LineReversalConnection) handleClientMessage() {

	currentLength := 0

	for clientMsg := range conn.sessionChan {
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
