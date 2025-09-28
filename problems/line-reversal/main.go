package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
)

func run() error {
	conn, err := net.ListenPacket("udp", ":8000")

	if err != nil {
		return err
	}

	defer conn.Close()

	log.Println("Listening for udp packages on :8000")

	var sessionTokenToChan = make(map[int]chan ClientMsg)

	for {
		var packet [1000]byte

		n, addr, err := conn.ReadFrom(packet[:])

		if err != nil {
			log.Println(err)
			return nil
		}

		log.Printf("Got new packet of size: %d", n)

		clientMsg, err := parsePacketData(string(packet[:n]))

		if err != nil {
			log.Printf("got invalid packet data: %v. Ignoring this packet\n", err)
			continue
		}

		switch msg := (clientMsg).(type) {
		case *ConnectMsg:
			sessionChan, ok := sessionTokenToChan[msg.SessionToken()]

			if !ok {
				sessionChan := make(chan ClientMsg)
				go handleSession(conn, sessionChan, addr)
				sessionTokenToChan[msg.SessionToken()] = sessionChan
				sessionChan <- clientMsg
				continue
			}

			sessionChan <- clientMsg
		default:
			sessionChan, ok := sessionTokenToChan[msg.SessionToken()]

			if !ok {
				// TODO: Send closeSession msg instead
				log.Printf("got msgType: %v, but there is no session for :%v. Therefore ignoring the packet\n", msg.Type(), msg.SessionToken())
				continue
			}

			sessionChan <- clientMsg
		}

	}

}

// Blocks the routine
func handleSession(packetConn net.PacketConn, sessionChan chan ClientMsg, add net.Addr) {
	currentLength := 0

	for clientMsg := range sessionChan {
		switch msg := clientMsg.(type) {
		case *ConnectMsg:
			ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: 0}

			if _, err := packetConn.WriteTo(ackMsg.toByte(), add); err != nil {
				// TODO: Figure out the correct error handling
				panic(err)
			}

			log.Printf("sent connect ack msg %+v", ackMsg)
			continue

		case *DataMsg:
			if currentLength < msg.pos {
				// We have missed a previous msg
				ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}

				if _, err := packetConn.WriteTo(ackMsg.toByte(), add); err != nil {
					// TODO: Figure out the correct error handling
					panic(err)
				}

				log.Printf("sent data ack msg %+v", ackMsg)
				continue
			}

			newData := msg.data[currentLength-msg.pos:]

			log.Printf("got data: %v", newData)
			currentLength += len(newData)

			ackMsg := AckMsg{sessionToken: msg.SessionToken(), length: currentLength}

			if _, err := packetConn.WriteTo(ackMsg.toByte(), add); err != nil {
				// TODO: Figure out the correct error handling
				panic(err)
			}

			log.Printf("sent data ack msg %+v", ackMsg)
			continue

		}

	}

}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type ParsePacketState string

const (
	ParseMsgType      ParsePacketState = "msgType"
	ParseSessionToken ParsePacketState = "sessionToken"
	ParserPos         ParsePacketState = "pos"
	ParseData         ParsePacketState = "data"
	ParseLength       ParsePacketState = "length"
	EOF               ParsePacketState = "eof" // No more bytes are expected to be there
)

type PossibleMsgType string

const (
	ConnectMsgType PossibleMsgType = "connect"
	DataMsgType    PossibleMsgType = "data"
	AckMsgType     PossibleMsgType = "ack"
	CloseMsgType   PossibleMsgType = "close"
)

func parsePacketData(packetData string) (ClientMsg, error) {

	log.Printf("parsing packetData: %v", packetData)

	if len(packetData) == 0 {
		return nil, fmt.Errorf("packet data is empty")
	}

	state := ParseMsgType
	isSkipCharacter := false

	msgTypeBuilder := ""
	sessionTokenBuilder := ""
	posBuilder := ""
	dataBuilder := ""

	var msgType PossibleMsgType
	var sessionToken *int
	var pos *int
	var data *string

	for i, ch := range packetData {
		if state == EOF {
			return nil, fmt.Errorf("got character %v after state reached EOF", string(ch))
		}

		if i == 0 && ch != '/' {
			return nil, fmt.Errorf("invalid packet data. packet data must start with '/'")
		}

		if i == len(packetData)-1 && ch != '/' {
			return nil, fmt.Errorf("invalid packet data. packet data must end with '/'")
		}

		if i == 0 {
			continue
		}

		if isSkipCharacter {
			isSkipCharacter = false
			continue
		}

		if state == ParseMsgType {

			if ch == '/' {
				if len(msgTypeBuilder) == 0 {
					return nil, fmt.Errorf("invalid packet data. message type is empty")
				}

				switch msgTypeBuilder {
				case string(ConnectMsgType):
					msgType = ConnectMsgType
				case string(DataMsgType):
					msgType = DataMsgType
				case string(AckMsgType):
					msgType = AckMsgType
				case string(CloseMsgType):
					msgType = CloseMsgType
				default:
					return nil, fmt.Errorf("unknown msgType: %v", msgTypeBuilder)
				}

				state = ParseSessionToken
				continue
			}

			msgTypeBuilder += string(ch)
			continue
		}

		if state == ParseSessionToken {

			if ch == '/' {
				if len(sessionTokenBuilder) == 0 {
					return nil, fmt.Errorf("invalid packet data. session token is empty")
				}

				mightBeSessionToken, err := strconv.Atoi(sessionTokenBuilder)

				if err != nil {
					return nil, err
				}

				if mightBeSessionToken < 0 {
					return nil, fmt.Errorf("sessionToken has to be >= 0, instead it is %d", mightBeSessionToken)
				}

				sessionToken = &mightBeSessionToken

				switch msgType {
				case ConnectMsgType:
					state = EOF
				case DataMsgType:
					state = ParserPos
				case AckMsgType:
					state = ParseLength
				case CloseMsgType:
					state = EOF
				}
				continue
			}

			sessionTokenBuilder += string(ch)
			continue
		}

		if state == ParserPos {
			if msgType != DataMsgType {
				panic(fmt.Sprintf("state=%v, this can only happen if msgType=%v . But current msgType is %v", ParserPos, DataMsgType, msgType))
			}

			if ch == '/' {
				if len(posBuilder) == 0 {
					return nil, fmt.Errorf("invalid packet data. pos is empty")
				}

				maybeValidPos, err := strconv.Atoi(posBuilder)

				if err != nil {
					return nil, fmt.Errorf("invalid packet data, unable to convert the pos:%v to int. Got error %v", posBuilder, err)
				}

				if maybeValidPos < 0 {
					return nil, fmt.Errorf("invalid packat data, pos=%v is less than 0", maybeValidPos)
				}

				pos = &maybeValidPos

				state = ParseData
				continue
			}

			posBuilder += string(ch)
			continue
		}

		if state == ParseData {

			if ch == '/' {
				if len(dataBuilder) == 0 {
					return nil, fmt.Errorf("invalid packet data, data is empty")
				}

				data = &dataBuilder
				state = EOF
				continue
			}

			if ch == '\\' {
				nextChar := packetData[i+1]

				switch nextChar {
				case '/':
					dataBuilder += "/"
					isSkipCharacter = true
				case '\\':
					dataBuilder += "\\"
					isSkipCharacter = true
				default:
					return nil, fmt.Errorf("expected nextChar of \\ to be either / or \\ but instead got %s", string(nextChar))
				}
			}

			dataBuilder += string(ch)
			continue

		}

		panic("TODO")
	}

	if state != EOF {
		return nil, fmt.Errorf("invalid packet data, packet data is completed without reaching EOF state. Current state: %v", state)
	}

	if msgType == ConnectMsgType {
		if sessionToken == nil {
			return nil, fmt.Errorf("msgType = %v. But sessionToken is nil", msgType)
		}

		return &ConnectMsg{sessionToken: *sessionToken}, nil
	}

	if msgType == DataMsgType {
		if sessionToken == nil {
			return nil, fmt.Errorf("msgType = %v. But sessionToken is nil", msgType)
		}

		if pos == nil {
			return nil, fmt.Errorf("msgType = %v. But pos is nil", msgType)
		}

		if data == nil {
			return nil, fmt.Errorf("msgType = %v. But data is nil", msgType)
		}

		return &DataMsg{sessionToken: *sessionToken, pos: *pos, data: *data}, nil
	}

	panic("TODO")
}

type ClientMsg interface {
	SessionToken() int
	Type() PossibleMsgType
}

type ConnectMsg struct {
	sessionToken int
}

func (c *ConnectMsg) SessionToken() int {
	return c.sessionToken
}

func (c *ConnectMsg) Type() PossibleMsgType {
	return ConnectMsgType
}

type AckMsg struct {
	sessionToken int
	length       int
}

func (a *AckMsg) SessionToken() int {
	return a.sessionToken
}

func (a *AckMsg) Type() PossibleMsgType {
	return AckMsgType
}

func (a *AckMsg) toByte() []byte {
	stringFmt := fmt.Sprintf("/ack/%d/%d", a.sessionToken, a.length)

	return []byte(stringFmt)
}

type DataMsg struct {
	sessionToken int
	pos          int
	data         string
}

func (d *DataMsg) SessionToken() int {
	return d.sessionToken
}

func (d *DataMsg) Type() PossibleMsgType {
	return DataMsgType
}

func (d *DataMsg) toByte() []byte {
	stringFmt := fmt.Sprintf("/data/%d/%d/%s", d.sessionToken, d.pos, d.data)

	return []byte(stringFmt)
}
