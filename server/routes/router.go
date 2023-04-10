package routes

import (
	"github.com/CaioBittencourt/arvore-genealogica/service"
	"github.com/gin-gonic/gin"
)

func SetupRouter(personService service.PersonService) *gin.Engine {
	router := gin.Default()
	RegisterPersonRoutes(router, personService)

	return router
}
