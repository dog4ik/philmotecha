package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/swaggo/http-swagger"
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

	env_port, present := os.LookupEnv("PORT")
	if !present {
		log.Fatalf("PORT env variable is not present")
	}

	port, err := strconv.Atoi(env_port)

	if err != nil {
		log.Fatalf("Failed to convert env port to number")
	}

	conn := retry_db_connection(ctx, database_url)
	defer conn.Close(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /swagger/doc.json", swagger_config)
	mux.HandleFunc("GET /swagger/*", httpSwagger.Handler(httpSwagger.URL("http://localhost:6969/swagger/doc.json")))

	log.Printf("Started Listening on port %d", port)

	err = http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", port), mux)
	log.Fatalf("Failed to listen and serve")
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

func swagger_config(w http.ResponseWriter, r *http.Request) {
	config, err := os.ReadFile("docs/swagger.json")
	if err != nil {
		log.Fatalf("could not read swagger config")
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(config)
}
