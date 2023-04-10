package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	docs "github.com/CaioBittencourt/arvore-genealogica/docs"
	"github.com/CaioBittencourt/arvore-genealogica/repository/mongodb"
	"github.com/CaioBittencourt/arvore-genealogica/server/routes"
	"github.com/CaioBittencourt/arvore-genealogica/service"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	mongoClient := mongodb.MongoConn(os.Getenv("MONGO_URI"))
	defer mongoClient.Disconnect(context.Background())

	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personService := service.NewPersonService(personRepository)

	router := routes.SetupRouter(personService)

	docs.SwaggerInfo.BasePath = "/person"
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

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

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	<-shutdown
	if err := srv.Shutdown(context.Background()); err != nil {
		log.Fatal("Error on server shutdown:", err)
	}
}
