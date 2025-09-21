package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"

	_ "embed"

	"github.com/nivekithan/go-network/problems/speed-daemon/db"
	_ "modernc.org/sqlite"
)

var plateObservationChan chan int64

// Blocks the current goroutine
func processPlateObservation(queries *db.Queries) {
	ctx := context.Background()
	log.Println("Processing plate observation")

	for observationId := range plateObservationChan {
		log.Printf("Processing observation ID: %d\n", observationId)
		observation, err := queries.GetObservationById(ctx, observationId)

		if err != nil {
			log.Printf("Error getting observation by ID: %v\n", err)
			continue
		}

		road, err := queries.GetRoad(ctx, observation.RoadID)

		if err != nil {
			log.Printf("Error getting road by ID: %v\n", err)
			continue
		}

		accepetedSpeedLimit := road.SpeedLimit

		previousObservation, err := queries.GetPreviousObservation(ctx, db.GetPreviousObservationParams{
			PlateNumber: observation.PlateNumber,
			RoadID:      observation.RoadID,
			Timestamp:   observation.Timestamp,
		})

		if err == nil {
			distance := float64(observation.Location - previousObservation.Location)
			time := float64(observation.Timestamp-previousObservation.Timestamp) / (60 * 60)

			previousSpeedLimitFloat := (distance) / time
			previousSpeedLimit := int64(math.Round(previousSpeedLimitFloat))

			log.Printf("Speed limit %v", previousSpeedLimit)

			if previousSpeedLimit > accepetedSpeedLimit {
				log.Printf("Speed limit exceeded for plate %v on road %v\n", observation.PlateNumber, road.ID)
				continue
			}
		}

		nextObservation, err := queries.GetNextObservation(ctx, db.GetNextObservationParams{
			PlateNumber: observation.PlateNumber,
			RoadID:      observation.RoadID,
			Timestamp:   observation.Timestamp,
		})

		if err == nil {
			distance := float64(observation.Location - nextObservation.Location)
			time := float64(observation.Timestamp-nextObservation.Timestamp) / (60 * 60)

			nextSpeedLimitFloat := (distance) / time
			nextSpeedLimit := int64(math.Round(nextSpeedLimitFloat))

			log.Printf("Speed limit %v", nextSpeedLimit)
			if nextSpeedLimit > accepetedSpeedLimit {
				log.Printf("Speed limit exceeded for plate %v on road %v\n", observation.PlateNumber, road.ID)
				continue
			}
		}

		log.Printf("Speed limit not exceeded for plate %v on road %v\n", observation.PlateNumber, road.ID)
	}
}

func handleConnectionImpl(queries *db.Queries, conn net.Conn) error {

	reader := bufio.NewReader(conn)

	var messageType uint8

	if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
		return err
	}

	ctx := context.Background()

	switch messageType {
	case 0x80:
		log.Println("MessageType=IamCamera")
		camera, err := NewIamCamera(reader)

		if err != nil {
			return err
		}

		log.Printf("%+v\n", camera)

		if err := camera.Register(ctx, queries); err != nil {
			return err
		}

		isWantHeartbeat := false

		for {
			var messageType uint8

			if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
				return err
			}

			switch messageType {
			case 0x40:
				log.Println("MessageType=WantHeartbeat")

				if isWantHeartbeat {
					return clientError(conn, fmt.Sprintf("Multiple heartbeats not allowed: %x", messageType))
				}

				isWantHeartbeat = true

				hearbeat, err := NewWantHeartbeat(reader)

				if err != nil {
					return err
				}

				go hearbeat.SendHeartBeat(conn)

			case 0x20:
				log.Println("MessageType=Plate")

				plate, err := NewPlate(reader)

				if err != nil {
					return err
				}

				log.Printf("%+v\n", plate)

				observation_id, err := plate.RegisterObservation(ctx, queries, RegisterObservationsParams{
					RoadID:   camera.road,
					Location: camera.mile,
				})

				if err != nil {
					return err
				}

				plateObservationChan <- observation_id

			default:
				return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
			}
		}

	case 0x81:
		log.Println("MessageType=IamDispatcher")
		camera, err := NewIamDispatcher(reader)

		if err != nil {
			return err
		}
		log.Printf("%+v\n", camera)

		isWantHeartbeat := false

		for {

			var messageType uint8

			if err := binary.Read(reader, binary.BigEndian, &messageType); err != nil {
				return err
			}

			switch messageType {

			case 0x40:
				log.Println("MessageType=WantHeartbeat")

				if isWantHeartbeat {
					return clientError(conn, fmt.Sprintf("Multiple heartbeats not allowed: %x", messageType))
				}

				isWantHeartbeat = true

				hearbeat, err := NewWantHeartbeat(reader)

				if err != nil {
					return err
				}

				go hearbeat.SendHeartBeat(conn)

			default:
				return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
			}
		}

	default:
		return clientError(conn, fmt.Sprintf("unknown messageType: %x", messageType))
	}
}

// This function blocks
func handleConnection(queries *db.Queries, conn net.Conn) {
	defer conn.Close()
	defer log.Println("Closing connection")

	if err := handleConnectionImpl(queries, conn); err != nil {
		log.Println("error: ", err)
	}
}

func handleListner(queries *db.Queries, listner net.Listener) error {
	conn, err := listner.Accept()

	if err != nil {
		return err
	}

	go handleConnection(queries, conn)

	return nil
}

//go:embed sql/schema.sql
var ddl string

func run() error {

	ctx := context.Background()
	sqliteDb, err := sql.Open("sqlite", "file::memory:?cache=shared")

	if err != nil {
		return err
	}

	log.Println(ddl)

	if _, err := sqliteDb.ExecContext(ctx, ddl); err != nil {
		return err
	}

	queries := db.New(sqliteDb)

	plateObservationChan = make(chan int64)

	go processPlateObservation(queries)

	listner, err := net.Listen("tcp", ":8000")

	if err != nil {
		return err
	}

	log.Println("Listening in port :8000")

	for {

		err := handleListner(queries, listner)

		if err != nil {
			log.Println("error: handleListner", err)
		}

	}

}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func clientError(conn net.Conn, msg string) error {
	clientError := ClientError{msg: msg}

	binary := clientError.toBinary()

	if _, err := conn.Write(binary); err != nil {
		return err
	}

	return errors.New(clientError.msg)
}
