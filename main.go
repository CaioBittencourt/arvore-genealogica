package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/repository/mongodb"
	"github.com/CaioBittencourt/arvore-genealogica/server/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	err := godotenv.Load(".env")
	// /etc/.env
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	mongoClient := mongoConn(os.Getenv("MONGO_URI"))
	defer mongoClient.Disconnect(context.Background())

	personRepository := mongodb.NewPersonRepository(*mongoClient)
	personController := controller.NewPersonController(personRepository)

	router := gin.Default()
	routes.RegisterPersonRoutes(router, personController)

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

func mongoConn(mongoURI string) *mongo.Client {
	client, err := mongo.NewClient(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}

	_, err = client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		log.Fatal(err)
	}

	return client
}
