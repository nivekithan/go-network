package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"

	_ "embed"

	"github.com/nivekithan/go-network/problems/means-to-end/db"
	_ "modernc.org/sqlite"
)

func handleConn(conn net.Conn, queries *db.Queries) {
	defer conn.Close()
	connectionId := rand.Text()

	ctx := context.Background()

	for {
		var encodedCommand [9]byte

		if _, err := io.ReadFull(conn, encodedCommand[:]); err != nil {
			if err != io.EOF {
				log.Printf("error reading command: %v, connection id: %s \n", err, connectionId)
				return
			}

			log.Printf("got EOF from client. Closing connection: %s\n", connectionId)
			return
		}

		command, err := parseCommand(encodedCommand, connectionId)

		if err != nil {
			log.Println(err)
			return
		}

		switch parsedCommand := command.(type) {
		case *InsertCommand:
			if err := queries.InsertAssestPrice(ctx, db.InsertAssestPriceParams{
				ID:        rand.Text(),
				AssestID:  connectionId,
				Timestamp: int64(parsedCommand.Timestamp),
				Price:     int64(parsedCommand.Price),
			}); err != nil {
				log.Println(err)
				return
			}

		case *QueryCommand:
			prices, err := queries.GetAssestPriceInTimeRange(ctx, db.GetAssestPriceInTimeRangeParams{
				AssestID:     connectionId,
				MinTimestamp: int64(parsedCommand.MinTime),
				MaxTimestamp: int64(parsedCommand.MaxTime),
			})

			if err != nil {
				log.Println(err)
				return
			}

			mean := findMean(prices)

			log.Printf("mean: %d", mean)

			var result [4]byte

			_, err = binary.Encode(result[:], binary.BigEndian, int32(mean))

			if err != nil {
				log.Println(err)
				return
			}

			_, err = conn.Write(result[:])

			if err != nil {
				log.Println(err)
				return
			}
		}

	}
}

func findMean(prices []int64) int {
	if len(prices) == 0 {
		return 0
	}

	sum := int(0)

	for _, curPrice := range prices {
		sum += int(curPrice)
	}

	mean := sum / len(prices)

	return mean
}

type Command interface {
	Type() string
}

type InsertCommand struct {
	Timestamp int32
	Price     int32
}

func (c *InsertCommand) Type() string {
	return "I"
}

type QueryCommand struct {
	MinTime int32
	MaxTime int32
}

func (c *QueryCommand) Type() string {
	return "Q"
}

func parseCommand(command [9]byte, connectionId string) (Command, error) {

	commandInstruction := string(command[0:1])

	if commandInstruction != "I" && commandInstruction != "Q" {
		return nil, fmt.Errorf("Invalid command instruction %s. Closing connection: %s", commandInstruction, connectionId)
	}

	log.Printf("Got command: %s from connection: %s\n", commandInstruction, connectionId)

	var firstInt32 int32
	firstInt32Reader := bytes.NewReader(command[1:])

	if err := binary.Read(firstInt32Reader, binary.BigEndian, &firstInt32); err != nil {
		return nil, fmt.Errorf("Error while reading firstInt32: %s", err)
	}

	log.Printf("firstInt32: %d\n", firstInt32)

	var secondInt32 int32

	if err := binary.Read(firstInt32Reader, binary.BigEndian, &secondInt32); err != nil {
		return nil, fmt.Errorf("Error while reading secondInt32: %s", err)
	}

	log.Printf("secondInt32: %d\n", secondInt32)

	if commandInstruction == "I" {
		return &InsertCommand{
			Timestamp: firstInt32,
			Price:     secondInt32,
		}, nil
	}

	return &QueryCommand{
		MinTime: firstInt32,
		MaxTime: secondInt32,
	}, nil
}

//go:embed sql/schema.sql
var ddl string

func run() error {

	ctx := context.Background()
	sqliteDb, err := sql.Open("sqlite", ":memory:")

	if err != nil {
		return err
	}

	if _, err := sqliteDb.ExecContext(ctx, ddl); err != nil {
		return err
	}

	queries := db.New(sqliteDb)

	listener, err := net.Listen("tcp", ":8000")

	log.Println("Listening on port 8000")

	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()

		if err != nil {
			return err
		}

		go handleConn(conn, queries)
	}

}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
