package routes

import (
	"github.com/CaioBittencourt/arvore-genealogica/server"
	"github.com/CaioBittencourt/arvore-genealogica/service"
	"github.com/gin-gonic/gin"
)

func RegisterPersonRoutes(router *gin.Engine, personService service.PersonService) {
	router.POST("/person", server.Store(personService))
	router.GET("/person/:id/tree", server.GetPersonFamilyRelationships(personService))
	router.GET("/person/:id/relationship/:id2", server.GetRelationshipBetweenPersons(personService))
	router.GET("/person/:id/baconNumber/:id2", server.GetBaconsNumberBetweenTwoPersons(personService))
}
