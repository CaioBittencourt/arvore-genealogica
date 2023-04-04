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
	MotherID    *string  `json:"motherId"`
	FatherID    *string  `json:"fatherId`
	ChildrenIDs []string `json:"childrenIds"`
}

type GetBaconsNumberBetweenTwoPersonsResponse struct {
	Persons      []PersonResponse `json:"persons"`
	BaconsNumber uint             `json:"baconsNumber"`
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

func GetBaconsNumberBetweenTwoPersons(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personAID := ctx.Param("id")
		personBID := ctx.Param("personIdB")

		persons, baconsNumber, err := personController.BaconsNumber(ctx, personAID, personBID)
		if err != nil {
			ctx.JSON(500, err) // fazer error handling / error matching da camada de dominio com a do http
			return
		}

		if baconsNumber == nil {
			ctx.JSON(404, "unable to find bacons number between this two persons")
			return
		}

		ctx.JSON(200, GetBaconsNumberBetweenTwoPersonsResponse{Persons: buildPersonResponsesFromDomainPersons(persons), BaconsNumber: *baconsNumber})
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
		for _, childrenID := range personReq.ChildrenIDs {
			person.Children = append(person.Children, &domain.Person{ID: childrenID})
		}
	}

	return person

}
func buildSimplePersonResponseFromDomainPerson(domainPerson domain.Person) PersonResponse {
	return PersonResponse{
		ID:       domainPerson.ID,
		Name:     domainPerson.Name,
		Gender:   string(domainPerson.Gender),
		Children: []PersonResponse{},
		Parents:  []PersonResponse{},
	}
}

func buildPersonResponseFromDomainPerson(domainPerson domain.Person) PersonResponse {
	personResponse := buildSimplePersonResponseFromDomainPerson(domainPerson)

	for _, children := range domainPerson.Children {
		if children.Name == "" {
			continue
		}

		domainChildren := buildSimplePersonResponseFromDomainPerson(*children)
		personResponse.Children = append(personResponse.Children, domainChildren)
	}

	for _, parent := range domainPerson.Parents {
		if parent.Name == "" {
			continue
		}

		domainParent := buildSimplePersonResponseFromDomainPerson(*parent)
		personResponse.Parents = append(personResponse.Parents, domainParent)
	}

	return personResponse
}

func buildPersonResponsesFromDomainPersons(domainPersons []domain.Person) []PersonResponse {
	var personResponses []PersonResponse
	for _, domainPerson := range domainPersons {
		personResponses = append(personResponses, buildPersonResponseFromDomainPerson(domainPerson))
	}

	return personResponses
}

// TODO: fix response parents and children for this endpoint
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
