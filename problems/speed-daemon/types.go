package main

import (
	"context"
	"encoding/binary"
	"io"
	"log"
	"net"
	"time"

	"github.com/nivekithan/go-network/problems/speed-daemon/db"
)

type ClientError struct {
	msg string
}

func (e *ClientError) toBinary() []byte {
	output := []byte{byte(0x10)}
	output = append(output, byte(len(e.msg)))
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

type RegisterObservationsParams struct {
	RoadID   uint16
	Location uint16
}

func (p *Plate) RegisterObservation(ctx context.Context, queries *db.Queries, params RegisterObservationsParams) (int64, error) {
	observation_id, err := queries.InsertPlateObservation(ctx, db.InsertPlateObservationParams{
		PlateNumber: p.plate,
		RoadID:      int64(params.RoadID),
		Timestamp:   int64(p.timestamp),
		Location:    int64(params.Location),
	})

	if err != nil {
		return 0, err
	}

	log.Printf("Registered plate observations %+v, on %+v", p, params)

	return observation_id, nil
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

func (t *Ticket) toBinary() []byte {
	output := []byte{0x21, byte(len(t.plate))}

	output = append(output, []byte(t.plate)...)
	output, err := binary.Append(output, binary.BigEndian, t.road)

	if err != nil {
		panic(err)
	}

	output, err = binary.Append(output, binary.BigEndian, t.mile1)

	if err != nil {
		panic(err)
	}

	output, err = binary.Append(output, binary.BigEndian, t.timestamp1)

	if err != nil {
		panic(err)
	}

	output, err = binary.Append(output, binary.BigEndian, t.mile2)

	if err != nil {
		panic(err)
	}

	output, err = binary.Append(output, binary.BigEndian, t.timestamp2)

	if err != nil {
		panic(err)
	}

	output, err = binary.Append(output, binary.BigEndian, t.speed)

	if err != nil {
		panic(err)
	}

	return output
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

func (camera *IAmCamera) Register(ctx context.Context, queries *db.Queries) error {

	roadId := int64(camera.road)

	_, err := queries.GetRoad(ctx, roadId)

	if err != nil {
		if err := queries.InsertRoad(ctx, db.InsertRoadParams{
			ID:         roadId,
			SpeedLimit: int64(camera.limit),
		}); err != nil {
			return err
		}

		log.Printf("Registered new road: %d", roadId)
		return nil
	}

	log.Printf("Road already registered: %d", roadId)

	return nil
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

func (d *IamDispatcher) Register(ctx context.Context, queries *db.Queries, dispatcherId string) {
	for _, road := range d.roads {
		roadId := int64(road)

		if err := queries.AddDispatcherForRoad(ctx, db.AddDispatcherForRoadParams{
			RoadID:       roadId,
			DispatcherID: dispatcherId,
		}); err != nil {
			log.Printf("Failed to register dispatcher %s for road %d: %v", dispatcherId, roadId, err)
		}
	}
}
