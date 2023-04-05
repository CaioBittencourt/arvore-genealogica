package routes

import (
	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/gin-gonic/gin"
)

func SetupRouter(personController controller.PersonController) *gin.Engine {
	router := gin.Default()
	RegisterPersonRoutes(router, personController)

	return router
}
