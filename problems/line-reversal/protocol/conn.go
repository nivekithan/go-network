package protocol

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type PendingTransmission struct {
	pos    int
	data   string
	sentAt time.Time
	timer  *time.Timer
}

type LineReversalConnection struct {
	lis                  *LineReversalListener
	sessionChan          chan ClientMsg
	sessionOutgoingAddr  net.Addr
	sessionToken         int
	isClosed             bool
	bufferData           []byte
	bufferDataMutex      sync.Mutex
	bufferDataAvaliable  sync.Cond
	sentData             []byte
	ackLength            int
	pendingTransmissions []PendingTransmission
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
					ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}

					conn.writeToRemote(ackMsg.toByte())

					log.Printf("sent data ack msg %+v", ackMsg)
					continue
				}

				dataEndPos := msg.pos + len(msg.data)

				if currentLength >= dataEndPos {
					ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}
					conn.writeToRemote(ackMsg.toByte())
					log.Printf("sent data ack msg %+v (duplicate data)", ackMsg)
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
			case *AckMsg:
				conn.handleAckMsg(msg)
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

func (conn *LineReversalConnection) Write(b []byte) (int, error) {
	conn.bufferDataMutex.Lock()
	defer conn.bufferDataMutex.Unlock()

	if conn.isClosed {
		return 0, io.ErrClosedPipe
	}

	startPos := len(conn.sentData)
	conn.sentData = append(conn.sentData, b...)

	conn.sendUnacknowledgedData(startPos)

	return len(b), nil
}

func (conn *LineReversalConnection) sendUnacknowledgedData(startPos int) {
	newData := conn.sentData[startPos:]

	if len(newData) == 0 {
		return
	}

	escapedData := escapeData(string(newData))

	pos := startPos
	overhead := len(fmt.Sprintf("/data/%d/%d//", conn.sessionToken, pos))
	maxDataSize := 1000 - overhead - 10

	offset := 0
	for offset < len(escapedData) {
		chunkSize := maxDataSize
		if offset+chunkSize > len(escapedData) {
			chunkSize = len(escapedData) - offset
		}

		chunk := escapedData[offset : offset+chunkSize]
		dataMsg := DataMsg{
			sessionToken: conn.sessionToken,
			pos:          pos,
			data:         chunk,
		}

		conn.writeToRemote(dataMsg.toByte())
		log.Printf("sent data msg session=%d pos=%d len=%d", conn.sessionToken, pos, chunkSize)

		pending := PendingTransmission{
			pos:    pos,
			data:   chunk,
			sentAt: time.Now(),
		}

		timer := time.AfterFunc(3*time.Second, func() {
			conn.handleRetransmission(pending)
		})
		pending.timer = timer

		conn.pendingTransmissions = append(conn.pendingTransmissions, pending)

		unescapedChunkLen := len(unescapeData(chunk))
		offset += chunkSize
		pos += unescapedChunkLen
	}
}

func (conn *LineReversalConnection) handleRetransmission(pending PendingTransmission) {
	conn.bufferDataMutex.Lock()
	defer conn.bufferDataMutex.Unlock()

	if conn.isClosed {
		return
	}

	unescapedLen := len(unescapeData(pending.data))
	if pending.pos+unescapedLen <= conn.ackLength {
		return
	}

	dataMsg := DataMsg{
		sessionToken: conn.sessionToken,
		pos:          pending.pos,
		data:         pending.data,
	}

	conn.writeToRemote(dataMsg.toByte())
	log.Printf("retransmitted data msg session=%d pos=%d", conn.sessionToken, pending.pos)

	newTimer := time.AfterFunc(3*time.Second, func() {
		conn.handleRetransmission(PendingTransmission{
			pos:    pending.pos,
			data:   pending.data,
			sentAt: pending.sentAt,
		})
	})

	for i := range conn.pendingTransmissions {
		if conn.pendingTransmissions[i].pos == pending.pos {
			conn.pendingTransmissions[i].timer = newTimer
			break
		}
	}
}

func escapeData(data string) string {
	var result strings.Builder
	result.Grow(len(data) * 2)

	for _, ch := range data {
		if ch == '/' {
			result.WriteString("\\/")
		} else if ch == '\\' {
			result.WriteString("\\\\")
		} else {
			result.WriteRune(ch)
		}
	}

	return result.String()
}

func unescapeData(data string) string {
	var result strings.Builder
	result.Grow(len(data))

	i := 0
	for i < len(data) {
		if data[i] == '\\' && i+1 < len(data) {
			if data[i+1] == '/' {
				result.WriteByte('/')
				i += 2
				continue
			} else if data[i+1] == '\\' {
				result.WriteByte('\\')
				i += 2
				continue
			}
		}
		result.WriteByte(data[i])
		i++
	}

	return result.String()
}

func (conn *LineReversalConnection) handleAckMsg(msg *AckMsg) {
	conn.bufferDataMutex.Lock()

	if msg.length > len(conn.sentData) {
		log.Printf("Peer misbehaving: ack length %d > sent data %d, closing session %d", msg.length, len(conn.sentData), conn.sessionToken)

		for _, p := range conn.pendingTransmissions {
			p.timer.Stop()
		}
		conn.pendingTransmissions = []PendingTransmission{}

		closeMsg := CloseMsg{sessionToken: conn.sessionToken}
		conn.writeToRemote(closeMsg.toByte())

		conn.lis.closeSession(conn.sessionToken)
		conn.isClosed = true
		conn.bufferDataAvaliable.Broadcast()
		conn.bufferDataMutex.Unlock()
		return
	}

	defer conn.bufferDataMutex.Unlock()

	if msg.length <= conn.ackLength {
		log.Printf("Duplicate or old ack: ack length %d <= current ack %d, retransmitting", msg.length, conn.ackLength)
		conn.retransmitFrom(msg.length)
		return
	}

	conn.ackLength = msg.length
	log.Printf("Ack received: session=%d length=%d", conn.sessionToken, msg.length)

	newPending := []PendingTransmission{}
	for _, p := range conn.pendingTransmissions {
		unescapedLen := len(unescapeData(p.data))
		if p.pos+unescapedLen <= conn.ackLength {
			p.timer.Stop()
		} else {
			newPending = append(newPending, p)
		}
	}
	conn.pendingTransmissions = newPending

	if msg.length < len(conn.sentData) {
		log.Printf("Partial ack: retransmitting from %d", msg.length)
		conn.retransmitFrom(msg.length)
	}
}

func (conn *LineReversalConnection) retransmitFrom(pos int) {
	for _, p := range conn.pendingTransmissions {
		p.timer.Stop()
	}
	conn.pendingTransmissions = []PendingTransmission{}

	conn.sendUnacknowledgedData(pos)
}
