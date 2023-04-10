package server

import (
	"net/http"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/service"
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

func buildPersonsWithRelationshipFromPerson(person domain.Person) PersonWithRelationship {
	personWithRelationship := PersonWithRelationship{
		RelationshipPerson: RelationshipPerson{
			Name:   person.Name,
			ID:     person.ID,
			Gender: string(person.Gender),
		},
	}

	personWithRelationship.Relationships = []Relationship{}
	for _, relationship := range person.Relationships {
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

	return personWithRelationship
}

func buildPersonsWithRelationshipFromFamilyGraph(familyGraph domain.FamilyGraph) map[string]PersonWithRelationship {
	personsWithRelationship := map[string]PersonWithRelationship{}
	for _, member := range familyGraph.Members {
		memberWithRelationship := buildPersonsWithRelationshipFromPerson(*member)
		personsWithRelationship[member.ID] = memberWithRelationship
	}

	return personsWithRelationship
}

// @Summary      Get family tree with relationships for person
// @Description  Get family tree with relationships for person
// @Accept       json
// @Produce      json
// @Param        id   path       string  true  "Person ID"
// @Success      200  {object}   PersonTreeResponse
// @Failure      400  {object}   ErrorResponse
// @Failure      404  {object}   ErrorResponse
// @Failure      500  {object}   ErrorResponse
// @Router       /person/:id/tree [get]
func GetPersonFamilyRelationships(personService service.PersonService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personID := ctx.Param("id")

		familyGraph, err := personService.GetFamilyGraphByPersonID(ctx, personID)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, PersonTreeResponse{Members: buildPersonsWithRelationshipFromFamilyGraph(*familyGraph)})
	})
}

// @Summary      Get bacons number between two persons
// @Description  Get bacons number between two persons
// @Accept       json
// @Produce      json
// @Param        id   path       string  true  "Person ID"
// @Param        id2   path      string  true  "Person 2 ID"
// @Success      200  {object}   GetBaconsNumberBetweenTwoPersonsResponse
// @Failure      400  {object}   ErrorResponse
// @Failure      404  {object}   ErrorResponse
// @Failure      500  {object}   ErrorResponse
// @Router       /person/:id/baconNumber/:id2 [get]
func GetBaconsNumberBetweenTwoPersons(personService service.PersonService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personAID := ctx.Param("id")
		personBID := ctx.Param("id2")

		baconsNumber, err := personService.BaconsNumber(ctx, personAID, personBID)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, GetBaconsNumberBetweenTwoPersonsResponse{BaconsNumber: *baconsNumber})
	})
}

// @Summary      Get relationship between two persons
// @Description  Get relationship between two persons
// @Accept       json
// @Produce      json
// @Param        id   path       string  true  "Person ID"
// @Param        id2   path      string  true  "Person 2 ID"
// @Success      200  {object}   PersonWithRelationship
// @Failure      400  {object}   ErrorResponse
// @Failure      404  {object}   ErrorResponse
// @Failure      500  {object}   ErrorResponse
// @Router       /person/:id/relationship/:id2 [get]
func GetRelationshipBetweenPersons(personService service.PersonService) gin.HandlerFunc {
	return gin.HandlerFunc(func(ctx *gin.Context) {
		personAID := ctx.Param("id")
		personBID := ctx.Param("id2")

		personWithRelationship, err := personService.GetRelationshipBetweenPersons(ctx, personAID, personBID)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, buildPersonsWithRelationshipFromPerson(*personWithRelationship))
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

// @Summary      Store person
// @Description  Store person
// @Accept       json
// @Produce      json
// @Param        person   body       StorePersonRequest  true  "Person to store"
// @Success      200  {object}   PersonResponse
// @Failure      400  {object}   ErrorResponse
// @Failure      500  {object}   ErrorResponse
// @Router       /person [post]
func Store(personService service.PersonService) gin.HandlerFunc {
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
		person, err := personService.Store(ctx, personToStore)
		if err != nil {
			createServerResponseFromError(ctx, err)
			return
		}

		ctx.JSON(http.StatusOK, buildPersonResponseFromDomainPerson(*person))
	})
}
