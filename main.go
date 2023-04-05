package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/repository/mongodb"
	"github.com/CaioBittencourt/arvore-genealogica/server/routes"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	// /etc/.env
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	mongoClient := mongodb.MongoConn(os.Getenv("MONGO_URI"))
	defer mongoClient.Disconnect(context.Background())

	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personController := controller.NewPersonController(personRepository)

	router := routes.SetupRouter(personController)

	srv := &http.Server{
		Addr:    ":80",
		Handler: router,
	}

	go func() {
		// service connections
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal("Error on server shutdown:", err)
	}
}
