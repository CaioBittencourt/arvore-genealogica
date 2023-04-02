package routes

import (
	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/server"
	"github.com/gin-gonic/gin"
)

func RegisterPersonRoutes(router *gin.Engine, personController controller.PersonController) {
	router.GET("/person/:id/tree", server.GetPersonFamilyTreeHandler(personController))
	// router.GET("/person/:id/relationship", server.GetPersonFamilyTreeHandler(personController))
	router.POST("/person", server.Store(personController))
}
