package server

import (
	"errors"
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

type RelationshipPerson struct {
	ID     string
	Name   string
	Gender string
}

type Relationship struct {
	Person       RelationshipPerson
	Relationship string
}

type PersonWithRelationship struct {
	RelationshipPerson
	Relationships []Relationship
}
type PersonTreeResponse struct {
	Members []PersonWithRelationship
}

func GetPersonFamilyTreeHandler(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personID := ctx.Param("id")

		familyTree, err := personController.GetFamilyTreeByPersonID(ctx, personID)
		if err != nil {
			ctx.JSON(500, err) // fazer error handling / error matching da camada de dominio com a do http
			return
		}

		var resMembers []PersonWithRelationship
		for _, member := range familyTree.Members {
			personWithRelationship := PersonWithRelationship{
				RelationshipPerson: RelationshipPerson{
					Name:   member.Name,
					ID:     member.ID,
					Gender: string(member.Gender),
				},
			}

			for _, relationship := range member.Relationships {
				personWithRelationship.Relationships = append(
					personWithRelationship.Relationships,
					Relationship{
						Relationship: string(relationship.Relationship),
						Person: RelationshipPerson{
							Name:   relationship.Person.Name,
							ID:     relationship.Person.ID,
							Gender: string(relationship.Person.Gender),
						},
					})
			}

			resMembers = append(resMembers, personWithRelationship)
		}

		ctx.JSON(200, PersonTreeResponse{Members: resMembers})
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
		person.Parents = append(person.Parents, &domain.Person{ID: *personReq.FatherID})
	}

	if personReq.MotherID != nil {
		person.Parents = append(person.Parents, &domain.Person{ID: *personReq.MotherID})
	}

	if len(personReq.ChildrenIDs) > 0 {
		children := make([]*domain.Person, len(personReq.ChildrenIDs))
		for _, childrenID := range personReq.ChildrenIDs {

			children = append(children, &domain.Person{ID: childrenID})
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
