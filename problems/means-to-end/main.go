package main

import (
	"context"
	"database/sql"
	"log"

	_ "embed"

	"github.com/nivekithan/go-network/problems/means-to-end/db"
	_ "modernc.org/sqlite"
)

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

	assestPrice, err := queries.GetAllAssestsPrice(ctx)

	log.Println(assestPrice)

	return nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
