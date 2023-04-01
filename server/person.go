package server

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// make fields required
type StorePersonRequest struct {
	Name        string   `json:"name"`
	Gender      string   `json:"gender"`
	MotherID    *string  `json:"mother_id"`
	FatherID    *string  `json:"father_id"`
	ChildrenIDs []string `json:"children_ids"`
}

type PersonResponse struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Gender   string           `json:"gender"`
	Parents  []PersonResponse `json:"parents"`
	Children []PersonResponse `json:"children"`
}

func GetPersonFamilyTreeHandler(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personID := ctx.Param("id")

		person, err := personController.GetFamilyTreeByPersonID(ctx, personID)
		if err != nil {
			ctx.JSON(500, nil) // fazer error handling / error matching da camada de dominio com a do http
		}

		ctx.JSON(200, person)
	})
}
func (pr StorePersonRequest) validate() error {
	gender := domain.GenderType(pr.Gender)
	if len(pr.Name) < 2 {
		return errors.New("name must have more than 1 character")
	}

	if !gender.IsValid() {
		return errors.New("gender has to be male of female")
	}

	return nil
}

func buildPersonFromStorePersonRequest(personReq StorePersonRequest) domain.Person {
	person := domain.Person{Name: personReq.Name}

	person.Gender = domain.GenderType(personReq.Gender)

	if personReq.FatherID != nil {
		person.Parents = append(person.Parents, domain.Person{ID: *personReq.FatherID})
	}

	if personReq.MotherID != nil {
		person.Parents = append(person.Parents, domain.Person{ID: *personReq.MotherID})
	}

	if len(personReq.ChildrenIDs) > 0 {
		children := make([]domain.Person, len(personReq.ChildrenIDs))
		for _, childrenID := range personReq.ChildrenIDs {

			children = append(children, domain.Person{ID: childrenID})
		}

		person.Children = children
	}

	return person

}

// TODO add tests for required fields on api
func Store(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		var req StorePersonRequest

		if err := ctx.ShouldBindJSON(&req); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		if err := req.validate(); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
		}

		personToStore := buildPersonFromStorePersonRequest(req)
		fmt.Println(personToStore)
		person, err := personController.Store(ctx, personToStore)
		if err != nil {
			log.WithError(err).Error("error saving person")
			ctx.JSON(500, "failed to store person")
			return
		}

		//TODO: Maybe add some validation?

		ctx.JSON(200, person)
	})
}

// var personID string
