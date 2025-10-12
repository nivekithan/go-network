package protocol

import (
	"fmt"
	"log"
	"strconv"
)

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

func ParsePacketData(packetData string) (ClientMsg, error) {

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

	if msgType == CloseMsgType {
		if sessionToken == nil {
			return nil, fmt.Errorf("msgType = %v. But sessionToken is nil", msgType)
		}

		return &CloseMsg{sessionToken: *sessionToken}, nil
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
	stringFmt := fmt.Sprintf("/ack/%d/%d/", a.sessionToken, a.length)

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
	stringFmt := fmt.Sprintf("/data/%d/%d/%s/", d.sessionToken, d.pos, d.data)

	return []byte(stringFmt)
}

type CloseMsg struct {
	sessionToken int
}

func (c *CloseMsg) SessionToken() int {
	return c.sessionToken
}

func (c *CloseMsg) Type() PossibleMsgType {
	return CloseMsgType
}

func (c *CloseMsg) toByte() []byte {
	stringFmt := fmt.Sprintf("/close/%d/", c.sessionToken)

	return []byte(stringFmt)
}
