package main

import (
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"
)

type ClientError struct {
	msg string
}

func (e *ClientError) toBinary() []byte {
	output := []byte{byte(0x10)}
	output = append(output, []byte(e.msg)...)
	return output
}

type Plate struct {
	plate     string
	timestamp uint32
}

func NewPlate(reader io.Reader) (*Plate, error) {

	var plateLength uint8

	if err := binary.Read(reader, binary.BigEndian, &plateLength); err != nil {
		return nil, err
	}

	plateBytes := []byte{}

	for i := uint8(0); i < plateLength; i++ {
		var char byte
		if err := binary.Read(reader, binary.BigEndian, &char); err != nil {
			return nil, err
		}
		plateBytes = append(plateBytes, char)
	}

	plate := string(plateBytes)

	var timestamp uint32
	if err := binary.Read(reader, binary.BigEndian, &timestamp); err != nil {
		return nil, err
	}

	return &Plate{
		plate:     plate,
		timestamp: timestamp,
	}, nil

}

type Ticket struct {
	plate      string
	road       uint16
	mile1      uint16
	timestamp1 uint32
	mile2      uint16
	timestamp2 uint32

	// Speed is represented by 100 * miles / hour
	speed uint16
}

type WantHeartbeat struct {
	// Interval is represented in deciseconds
	// 25 deciseconds = 2.5 seconds
	interval uint32
}

func NewWantHeartbeat(reader io.Reader) (*WantHeartbeat, error) {
	var interval uint32

	if err := binary.Read(reader, binary.BigEndian, &interval); err != nil {
		return nil, err
	}

	return &WantHeartbeat{
		interval: interval,
	}, nil
}

// This function blocks
func (heartbeat *WantHeartbeat) SendHeartBeat(conn net.Conn) {

	if heartbeat.interval == 0 {
		log.Println("heartbeat interval is zero")
		return
	}

	interval := int(heartbeat.interval)

	timer := time.NewTicker(time.Duration(interval) * (time.Second / 10))

	for {
		<-timer.C

		heartbeat := &Heartbeat{}
		if _, err := conn.Write(heartbeat.toBinary()); err != nil {
			log.Println("error sending heartbeat:", err)
			log.Println("Stoppping Sendheart")
			return
		}

		log.Println("Sent hearbeat")
	}

}

type Heartbeat struct {
}

func (h *Heartbeat) toBinary() []byte {
	return []byte{0x41}

}

type IAmCamera struct {
	road  uint16
	mile  uint16
	limit uint16
}

func NewIamCamera(reader io.Reader) (*IAmCamera, error) {
	var road, mile, limit uint16

	if err := binary.Read(reader, binary.BigEndian, &road); err != nil {
		return nil, err
	}

	if err := binary.Read(reader, binary.BigEndian, &mile); err != nil {
		return nil, err
	}

	if err := binary.Read(reader, binary.BigEndian, &limit); err != nil {
		return nil, err
	}

	return &IAmCamera{
		road:  road,
		mile:  mile,
		limit: limit,
	}, nil
}

type IamDispatcher struct {
	roads []uint16
}

func NewIamDispatcher(reader io.Reader) (*IamDispatcher, error) {
	var numOfRoads uint8

	if err := binary.Read(reader, binary.BigEndian, &numOfRoads); err != nil {
		return nil, err
	}

	roads := []uint16{}

	for i := 0; i < int(numOfRoads); i++ {
		var road uint16

		if err := binary.Read(reader, binary.BigEndian, &road); err != nil {
			return nil, err
		}

		roads = append(roads, road)
	}

	return &IamDispatcher{
		roads: roads,
	}, nil
}
