package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"strconv"
	"time"

	_ "embed"

	"github.com/nivekithan/go-network/problems/speed-daemon/db"
	_ "modernc.org/sqlite"
)

var plateObservationChan chan int64
var dispatcherConnMap map[string]net.Conn

func processUnProcessedTicket(queries *db.Queries) {
	ctx := context.Background()
	ticker := time.NewTicker(1 * time.Second)

	for _ = range ticker.C {

		tickets, err := queries.GetUnProcessedTickets(ctx)
		if err != nil {
			log.Printf("Error getting unprocessed tickets: %v", err)
			return
		}

		for _, ticket := range tickets {
			dispatcherConn, ok := dispatcherConnMap[strconv.Itoa(int(ticket.RoadID))]

			if !ok {
				log.Printf("No dispatcher connection found for road %v", ticket.RoadID)
				continue
			}

			ticketBinary := Ticket{
				plate:      ticket.PlateNumber,
				road:       uint16(ticket.RoadID),
				mile1:      uint16(ticket.Mile1),
				timestamp1: uint32(ticket.Timestamp1),
				timestamp2: uint32(ticket.Timestamp2),
				mile2:      uint16(ticket.Mile2),
				speed:      uint16(ticket.Speed),
			}

			dispatcherConn.Write(ticketBinary.toBinary())

			queries.MarkTicketAsProcessed(ctx, ticket.ID)

		}
	}
}

type TicketObservation struct {
	timestamp int64
	location  int64
}

type CreateNewTicketParams struct {
	plate        string
	roadId       int64
	observation1 TicketObservation
	observation2 TicketObservation
	speed        int64
}

func createNewTicket(ctx context.Context, queries *db.Queries, newTicket CreateNewTicketParams) {
	var ticket Ticket

	if newTicket.observation1.timestamp > newTicket.observation2.timestamp {
		ticket = Ticket{
			plate:      newTicket.plate,
			road:       uint16(newTicket.roadId),
			mile1:      uint16(newTicket.observation2.location),
			mile2:      uint16(newTicket.observation1.location),
			timestamp1: uint32(newTicket.observation2.timestamp),
			timestamp2: uint32(newTicket.observation1.timestamp),
			speed:      uint16(newTicket.speed * 100),
		}
	} else {
		ticket = Ticket{
			plate:      newTicket.plate,
			road:       uint16(newTicket.roadId),
			mile1:      uint16(newTicket.observation1.location),
			mile2:      uint16(newTicket.observation2.location),
			timestamp1: uint32(newTicket.observation1.timestamp),
			timestamp2: uint32(newTicket.observation2.timestamp),
			speed:      uint16(newTicket.speed * 100),
		}
	}

	minDay := math.Trunc(float64(ticket.timestamp1 / 86400))
	maxDay := math.Trunc(float64(ticket.timestamp2 / 86400))

	if _, err := queries.ConflictingTickets(ctx, db.ConflictingTicketsParams{
		PlateNumber: ticket.plate,
		StartDate:   int64(minDay),
		EndDate:     int64(maxDay),
	}); err == nil {
		log.Println("Found conflicting tickets")
		return
	}

	log.Println("Did not find conflicting tickets. Ticketing the plate")

	dispatcherId, err := queries.FindDispatcherForRoad(ctx, int64(ticket.road))

	if err != nil {
		log.Printf("found no dispatcher for road: %v. Storing ticket as not processed", ticket.road)

		if err = queries.StoreTicket(ctx, db.StoreTicketParams{
			PlateNumber:   ticket.plate,
			RoadID:        int64(ticket.road),
			Mile1:         int64(ticket.mile1),
			Mile2:         int64(ticket.mile2),
			Timestamp1:    int64(ticket.timestamp1),
			Timestamp2:    int64(ticket.timestamp2),
			Speed:         int64(ticket.speed),
			DayStartRange: int64(minDay),
			DayEndRange:   int64(maxDay),
			IsProcessed:   0,
		}); err != nil {
			panic(err)
		}

		return
	}

	conn, ok := dispatcherConnMap[dispatcherId]

	if !ok {
		log.Printf("Found no dispatcher for road: %v, Storing ticket as not processed", ticket.road)
		if err = queries.StoreTicket(ctx, db.StoreTicketParams{
			PlateNumber:   ticket.plate,
			RoadID:        int64(ticket.road),
			Mile1:         int64(ticket.mile1),
			Mile2:         int64(ticket.mile2),
			Timestamp1:    int64(ticket.timestamp1),
			Timestamp2:    int64(ticket.timestamp2),
			Speed:         int64(ticket.speed),
			DayStartRange: int64(minDay),
			DayEndRange:   int64(maxDay),
			IsProcessed:   0,
		}); err != nil {
			panic(err)
		}

		return
	}

	conn.Write(ticket.toBinary())

	if err = queries.StoreTicket(ctx, db.StoreTicketParams{
		PlateNumber:   ticket.plate,
		RoadID:        int64(ticket.road),
		Mile1:         int64(ticket.mile1),
		Mile2:         int64(ticket.mile2),
		Timestamp1:    int64(ticket.timestamp1),
		Timestamp2:    int64(ticket.timestamp2),
		Speed:         int64(ticket.speed),
		DayStartRange: int64(minDay),
		DayEndRange:   int64(maxDay),
		IsProcessed:   1,
	}); err != nil {
		panic(err)
	}
}

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
			distance := math.Abs(float64(observation.Location - previousObservation.Location))
			time := math.Abs(float64(observation.Timestamp-previousObservation.Timestamp) / (60 * 60))

			previousSpeedLimitFloat := (distance) / time
			previousSpeedLimit := int64(math.Round(previousSpeedLimitFloat))

			log.Printf("Speed limit %v", previousSpeedLimit)

			if previousSpeedLimit > accepetedSpeedLimit {
				log.Printf("Speed limit exceeded for plate %v on road %v\n", observation.PlateNumber, road.ID)
				createNewTicket(ctx, queries, CreateNewTicketParams{
					plate:        observation.PlateNumber,
					roadId:       road.ID,
					observation1: TicketObservation{timestamp: observation.Timestamp, location: observation.Location},
					observation2: TicketObservation{timestamp: previousObservation.Timestamp, location: previousObservation.Location},
					speed:        previousSpeedLimit,
				})
				continue
			}

		}

		nextObservation, err := queries.GetNextObservation(ctx, db.GetNextObservationParams{
			PlateNumber: observation.PlateNumber,
			RoadID:      observation.RoadID,
			Timestamp:   observation.Timestamp,
		})

		if err == nil {
			distance := math.Abs(float64(observation.Location - nextObservation.Location))
			time := math.Abs(float64(observation.Timestamp-nextObservation.Timestamp) / (60 * 60))

			nextSpeedLimitFloat := (distance) / time
			nextSpeedLimit := int64(math.Round(nextSpeedLimitFloat))

			log.Printf("Speed limit %v", nextSpeedLimit)
			if nextSpeedLimit > accepetedSpeedLimit {
				log.Printf("Speed limit exceeded for plate %v on road %v\n", observation.PlateNumber, road.ID)
				createNewTicket(ctx, queries, CreateNewTicketParams{
					plate:        observation.PlateNumber,
					roadId:       road.ID,
					observation1: TicketObservation{timestamp: observation.Timestamp, location: observation.Location},
					observation2: TicketObservation{timestamp: nextObservation.Timestamp, location: nextObservation.Location},
					speed:        nextSpeedLimit,
				})
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
		dispatcher, err := NewIamDispatcher(reader)
		dispatcherId := rand.Text()

		if err != nil {
			return err
		}
		log.Printf("%+v\n", dispatcher)

		dispatcherConnMap[dispatcherId] = conn
		defer func() {
			delete(dispatcherConnMap, dispatcherId)
		}()
		dispatcher.Register(ctx, queries, dispatcherId)

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
	dispatcherConnMap = make(map[string]net.Conn)

	go processPlateObservation(queries)
	go processUnProcessedTicket(queries)

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
