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
	ID     string `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
}

type Relationship struct {
	Person       RelationshipPerson `json:"person"`
	Relationship string             `json:"relationship"`
}

type PersonWithRelationship struct {
	RelationshipPerson
	Relationships []Relationship `json:"relationships"`
}
type PersonTreeResponse struct {
	Members []PersonWithRelationship `json:"members"`
}

func buildPersonsWithRelationshipFromFamilyGraph(familyGraph domain.FamilyGraph) []PersonWithRelationship {
	var personsWithRelationship []PersonWithRelationship
	for _, member := range familyGraph.Members {
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

		personsWithRelationship = append(personsWithRelationship, personWithRelationship)
	}

	return personsWithRelationship
}

func GetPersonFamilyGraphHandler(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personID := ctx.Param("id")

		familyGraph, err := personController.GetFamilyGraphByPersonID(ctx, personID)
		if err != nil {
			ctx.JSON(500, err) // fazer error handling / error matching da camada de dominio com a do http
			return
		}

		ctx.JSON(200, PersonTreeResponse{Members: buildPersonsWithRelationshipFromFamilyGraph(*familyGraph)})
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

func buildPersonResponseFromDomainPerson(domainPerson domain.Person) PersonResponse {
	personResponse := &PersonResponse{
		ID:     domainPerson.ID,
		Name:   domainPerson.Name,
		Gender: string(domainPerson.Gender),
	}

	personResponse.Children = []PersonResponse{}
	for _, children := range domainPerson.Children {
		if children.Name == "" {
			continue
		}

		domainChildren := buildPersonResponseFromDomainPerson(*children)
		personResponse.Children = append(personResponse.Children, domainChildren)
	}

	personResponse.Parents = []PersonResponse{}
	for _, parent := range domainPerson.Parents {
		if parent.Name == "" {
			continue
		}

		domainParent := buildPersonResponseFromDomainPerson(*parent)
		personResponse.Parents = append(personResponse.Parents, domainParent)
	}

	return *personResponse
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

		//TODO: Add request validation
		ctx.JSON(200, buildPersonResponseFromDomainPerson(*person))
	})
}
