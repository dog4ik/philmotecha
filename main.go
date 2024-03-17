package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

//	@title						Philmotecha API
//	@version					1.0
//	@description				This is a simple movie library server.
//	@securityDefinitions.apiKey	JwtAuth
//	@in							header
//	@name						Authorization

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Printf("WARN: Error loading .env file\n")
	}

	ctx := context.Background()

	database_url, present := os.LookupEnv("DATABASE_URL")

	if !present {
		log.Fatalf("DATABASE_URL env variable is not present")
	}

	conn := retry_db_connection(ctx, database_url)
	defer conn.Close(ctx)

}

func retry_db_connection(ctx context.Context, database_url string) *pgx.Conn {
	for {
		conn, err := pgx.Connect(ctx, database_url)
		if err != nil {
			log.Printf("failed to connect to the database: %s RETRYING", err)
			time.Sleep(1 * time.Second)
		} else {
			return conn
		}
	}
}
