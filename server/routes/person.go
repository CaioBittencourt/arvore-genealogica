package routes

import (
	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/server"
	"github.com/gin-gonic/gin"
)

func RegisterPersonRoutes(router *gin.Engine, personController controller.PersonController) {
	router.GET("/person/:id/tree", server.GetPersonFamilyRelationships(personController))
	// router.GET("/person/:id/relationship", server.GetPersonFamilyGraphHandler(personController))
	router.GET("/person/:id/baconNumber/:personIdB", server.GetBaconsNumberBetweenTwoPersons(personController))
	router.POST("/person", server.Store(personController))
}
