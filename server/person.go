package server

import (
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
	BaconsNumber uint `json:"baconsNumber"`
}

type PersonRelativesResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
}
type PersonResponse struct {
	ID       string                    `json:"id"`
	Name     string                    `json:"name"`
	Gender   string                    `json:"gender"`
	Parents  []PersonRelativesResponse `json:"parents"`
	Children []PersonRelativesResponse `json:"children"`
	Spouses  []PersonRelativesResponse `json:"spouses"`
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
	Members map[string]PersonWithRelationship `json:"members"`
}

func createServerResponseFromError(ctx *gin.Context, err error) {
	errResponse := BuildErrorResponseFromError(err)
	ctx.JSON(errResponse.StatusCode, errResponse)
}

func buildPersonsWithRelationshipFromFamilyGraph(familyGraph domain.FamilyGraph) map[string]PersonWithRelationship {
	personsWithRelationship := map[string]PersonWithRelationship{}
	for _, member := range familyGraph.Members {
		personWithRelationship := PersonWithRelationship{
			RelationshipPerson: RelationshipPerson{
				Name:   member.Name,
				ID:     member.ID,
				Gender: string(member.Gender),
			},
		}

		personWithRelationship.Relationships = []Relationship{}
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

		personsWithRelationship[member.ID] = personWithRelationship
	}

	return personsWithRelationship
}

func GetPersonFamilyRelationships(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personID := ctx.Param("id")

		familyGraph, err := personController.GetFamilyGraphByPersonID(ctx, personID)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, PersonTreeResponse{Members: buildPersonsWithRelationshipFromFamilyGraph(*familyGraph)})
	})
}

func GetBaconsNumberBetweenTwoPersons(personController controller.PersonController) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personAID := ctx.Param("id")
		personBID := ctx.Param("id2")

		baconsNumber, err := personController.BaconsNumber(ctx, personAID, personBID)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, GetBaconsNumberBetweenTwoPersonsResponse{BaconsNumber: *baconsNumber})
	})
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
func buildPersonRelativesResponseFromDomainPerson(domainPerson domain.Person) PersonRelativesResponse {
	return PersonRelativesResponse{
		ID:     domainPerson.ID,
		Name:   domainPerson.Name,
		Gender: string(domainPerson.Gender),
	}
}

func buildPersonResponseFromDomainPerson(domainPerson domain.Person) PersonResponse {
	personResponse := PersonResponse{
		ID:       domainPerson.ID,
		Name:     domainPerson.Name,
		Gender:   string(domainPerson.Gender),
		Children: []PersonRelativesResponse{},
		Parents:  []PersonRelativesResponse{},
		Spouses:  []PersonRelativesResponse{},
	}

	for _, children := range domainPerson.Children {
		if children.Name == "" {
			continue
		}

		domainChildren := buildPersonRelativesResponseFromDomainPerson(*children)
		personResponse.Children = append(personResponse.Children, domainChildren)
	}

	for _, parent := range domainPerson.Parents {
		if parent.Name == "" {
			continue
		}

		domainParent := buildPersonRelativesResponseFromDomainPerson(*parent)
		personResponse.Parents = append(personResponse.Parents, domainParent)
	}

	for _, spouse := range domainPerson.Spouses {
		if spouse.Name == "" {
			continue
		}

		domainSpouse := buildPersonRelativesResponseFromDomainPerson(*spouse)
		personResponse.Spouses = append(personResponse.Spouses, domainSpouse)
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
			log.WithError(err).Error("server person: store: invalid request")
			ctx.JSON(http.StatusBadRequest, ErrorResponse{
				ErrorMessage: err.Error(),
			})
			return
		}

		personToStore := buildPersonFromStorePersonRequest(req)
		person, err := personController.Store(ctx, personToStore)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, buildPersonResponseFromDomainPerson(*person))
	})
}
