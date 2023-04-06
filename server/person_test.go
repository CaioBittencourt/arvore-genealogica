package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/CaioBittencourt/arvore-genealogica/controller"
	"github.com/CaioBittencourt/arvore-genealogica/repository/mongodb"
	"github.com/CaioBittencourt/arvore-genealogica/server"
	"github.com/CaioBittencourt/arvore-genealogica/server/routes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/mongo"
)

var mongoClient *mongo.Client

func TestMain(m *testing.M) {
	os.Setenv("MONGO_DATABASE", "familyTreeTest")

	mongoClient = mongodb.MongoConn("mongodb://mongo:27017")
	defer mongoClient.Disconnect(context.Background())

	retCode := m.Run()
	os.Exit(retCode)
}

func teardownTest() {
	mongoClient.Database(os.Getenv("MONGO_DATABASE")).Collection("person").Drop(context.Background())
}

// Equalize IDS so i can use assert.Equal. The id cannot be a request argument since its an mongodb object ID
func addPersonIDToExpectedPersonTreeResponse(expectedMembers []server.PersonWithRelationship, personInsertedIdByName map[string]string) []server.PersonWithRelationship {
	for i, member := range expectedMembers {
		expectedMembers[i].ID = personInsertedIdByName[member.Name]

		for _, relationship := range expectedMembers[i].Relationships {
			relationship.Person.ID = personInsertedIdByName[relationship.Person.Name]
		}
	}

	return expectedMembers
}

func addPersonIDToExpectedResponse(expected *server.PersonResponse, actual server.PersonResponse) {
	expected.ID = actual.ID

	for i, actualParent := range actual.Parents {
		expected.Parents[i].ID = actualParent.ID
	}

	for i, actualChildren := range actual.Children {
		expected.Children[i].ID = actualChildren.ID
	}

	for i, actualSpouse := range actual.Spouses {
		expected.Spouses[i].ID = actualSpouse.ID
	}
}

func doStorePersonRequest(router *gin.Engine, req server.StorePersonRequest) (*server.PersonResponse, *server.ErrorResponse, int, error) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(req)
	if err != nil {
		return nil, nil, 0, err
	}

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/person", &buf)
	router.ServeHTTP(w, httpReq)

	if w.Code > 200 {
		res := server.ErrorResponse{}
		err = json.Unmarshal(w.Body.Bytes(), &res)
		if err != nil {
			return nil, nil, w.Code, err
		}

		return nil, &res, w.Code, nil
	}

	res := server.PersonResponse{}
	err = json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		return nil, nil, w.Code, err
	}

	return &res, nil, w.Code, nil
}

func doGetPersonFamilyRelationshipsRequest(router *gin.Engine, personID string) (*server.PersonTreeResponse, *server.ErrorResponse, int, error) {
	var buf bytes.Buffer

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/person/%s/tree", personID), &buf)
	router.ServeHTTP(w, httpReq)

	if w.Code > 200 {
		res := server.ErrorResponse{}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		if err != nil {
			return nil, nil, w.Code, err
		}

		return nil, &res, w.Code, nil
	}

	res := server.PersonTreeResponse{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		return nil, nil, w.Code, err
	}

	return &res, nil, w.Code, nil
}

func TestStore(t *testing.T) {
	//NOTE: Leaving this settup per test in case any tests want to introduce a mock for controllers or repository.
	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personController := controller.NewPersonController(personRepository)
	router := routes.SetupRouter(personController)

	// map to get ids to build relationships once that person is inserted
	personInsertedIdByName := map[string]string{}

	type testArgs struct {
		testName              string
		personToStore         server.StorePersonRequest
		fatherName            string
		motherName            string
		childrenNames         []string
		expectedStatusCode    int
		expectedResponse      *server.PersonResponse
		expectedErrorResponse *server.ErrorResponse
	}

	alfredo := server.StorePersonRequest{
		Name:   "Alfredo",
		Gender: "male",
	}

	helena := server.StorePersonRequest{
		Name:   "Helena",
		Gender: "female",
	}

	dayse := server.StorePersonRequest{
		Name:   "Dayse",
		Gender: "female",
	}

	tests := []testArgs{
		{
			testName: "should return bad request when name has less than 2 characters",
			personToStore: server.StorePersonRequest{
				Name:   "a",
				Gender: "male",
			},
			expectedStatusCode:    400,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: "name must have more than 1 character"},
		},
		{
			testName: "should return bad request when gender is not valid",
			personToStore: server.StorePersonRequest{
				Name:   "Caio",
				Gender: "unexistingGender",
			},
			expectedStatusCode:    400,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: "gender has to be male of female"},
		},
		{
			testName:           "should store person without relationships",
			personToStore:      alfredo,
			expectedStatusCode: 200,
			expectedResponse: &server.PersonResponse{
				Name:     alfredo.Name,
				Gender:   alfredo.Gender,
				Children: []server.PersonRelativesResponse{},
				Parents:  []server.PersonRelativesResponse{},
				Spouses:  []server.PersonRelativesResponse{},
			},
		},
		{
			testName:           "should store person with parents",
			personToStore:      dayse,
			fatherName:         alfredo.Name,
			expectedStatusCode: 200,
			expectedResponse: &server.PersonResponse{
				Name:     dayse.Name,
				Gender:   dayse.Gender,
				Parents:  []server.PersonRelativesResponse{{Name: alfredo.Name, Gender: alfredo.Gender}},
				Children: []server.PersonRelativesResponse{},
				Spouses:  []server.PersonRelativesResponse{},
			},
		},
		{
			testName:           "should store person with children and identify spouse relationship",
			personToStore:      helena,
			childrenNames:      []string{dayse.Name},
			expectedStatusCode: 200,
			expectedResponse: &server.PersonResponse{
				Name:     helena.Name,
				Gender:   helena.Gender,
				Parents:  []server.PersonRelativesResponse{},
				Children: []server.PersonRelativesResponse{{Name: dayse.Name, Gender: dayse.Gender}},
				Spouses:  []server.PersonRelativesResponse{{Name: alfredo.Name, Gender: alfredo.Gender}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				//Add ids of previously inserted entities
				for _, childrenName := range tt.childrenNames {
					tt.personToStore.ChildrenIDs = append(tt.personToStore.ChildrenIDs, personInsertedIdByName[childrenName])
				}

				if tt.fatherName != "" {
					fatherId := personInsertedIdByName[tt.fatherName]
					tt.personToStore.FatherID = &fatherId
				}

				if tt.motherName != "" {
					motherId := personInsertedIdByName[tt.motherName]
					tt.personToStore.MotherID = &motherId
				}

				successRes, errorRes, statusCode, err := doStorePersonRequest(router, tt.personToStore)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, tt.expectedStatusCode, statusCode)

				if tt.expectedErrorResponse != nil {
					assert.Equal(t, tt.expectedErrorResponse, errorRes)
				}

				if tt.expectedResponse != nil {
					personInsertedIdByName[successRes.Name] = successRes.ID
					addPersonIDToExpectedResponse(tt.expectedResponse, *successRes)
					assert.Equal(t, tt.expectedResponse, successRes)
				}
			}
		}(tt))
	}

	teardownTest()
}

func buildFamily(router *gin.Engine, personInsertedIdByName map[string]string) error {
	if err := storePerson(router, server.StorePersonRequest{Name: "Tunico", Gender: "male"}, personInsertedIdByName); err != nil {
		return err
	}
	tunicoID := personInsertedIdByName["Tunico"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Luis", Gender: "male", FatherID: &tunicoID}, personInsertedIdByName); err != nil {
		return err
	}
	luisID := personInsertedIdByName["Luis"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Dayse", Gender: "female"}, personInsertedIdByName); err != nil {
		return err
	}
	dayseID := personInsertedIdByName["Dayse"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Caio", Gender: "male", FatherID: &luisID, MotherID: &dayseID}, personInsertedIdByName); err != nil {
		return err
	}

	if err := storePerson(router, server.StorePersonRequest{Name: "Claudia", Gender: "female", FatherID: &tunicoID}, personInsertedIdByName); err != nil {
		return err
	}
	claudiaID := personInsertedIdByName["Claudia"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Livia", Gender: "female", MotherID: &claudiaID}, personInsertedIdByName); err != nil {
		return err
	}

	if err := storePerson(router, server.StorePersonRequest{Name: "Vivian", Gender: "female", FatherID: &luisID, MotherID: &dayseID}, personInsertedIdByName); err != nil {
		return err
	}
	vivianID := personInsertedIdByName["Vivian"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Caio Regis", Gender: "male", FatherID: &luisID, MotherID: &dayseID}, personInsertedIdByName); err != nil {
		return err
	}
	caioRegisID := personInsertedIdByName["Caio Regis"]

	if err := storePerson(router, server.StorePersonRequest{Name: "Cauã", Gender: "male", FatherID: &caioRegisID, MotherID: &vivianID}, personInsertedIdByName); err != nil {
		return err
	}

	return nil
}

func storePerson(router *gin.Engine, req server.StorePersonRequest, personInsertedIdByName map[string]string) error {
	resSuccess, _, statusCode, err := doStorePersonRequest(router, req)
	if err != nil {
		return err
	}

	if statusCode != 200 {
		return fmt.Errorf("test: failed building family of one person. status code: %d", statusCode)
	}

	personInsertedIdByName[req.Name] = resSuccess.ID
	return nil
}

func TestGetPersonFamilyGraphHandler(t *testing.T) {
	//NOTE: Leaving this settup per test in case any tests want to introduce a mock for controllers or repository.
	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personController := controller.NewPersonController(personRepository)
	router := routes.SetupRouter(personController)

	type testArgs struct {
		testName              string
		buildFamilyTreeFunc   func() error
		personToSearchName    string
		expectedStatusCode    int
		expectedResponse      *server.PersonTreeResponse
		expectedErrorResponse *server.ErrorResponse
	}

	personInsertedIdByName := map[string]string{}
	tests := []testArgs{
		//TODO: do failures tests.
		{
			testName: "should return tree when there is just one node",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, personInsertedIdByName); err != nil {
					return err
				}
				return nil
			},
			personToSearchName: "Loner",
			expectedStatusCode: 200,
			expectedResponse: &server.PersonTreeResponse{Members: []server.PersonWithRelationship{
				{
					RelationshipPerson: server.RelationshipPerson{
						Name:   "Loner",
						Gender: "male",
					},
					Relationships: []server.Relationship{}},
			}},
		},
		// {
		// 	testName: "should return tree relationships: nephew, aunt, cousin, spouse, parent, children, sibling",
		// 	buildFamilyTreeFunc: func() error {
		// 		if err := buildFamily(router, personInsertedIdByName); err != nil {
		// 			return err
		// 		}
		// 		return nil
		// 	},
		// 	personToSearchName: "Caio",
		// 	expectedStatusCode: 200,
		// 	expectedResponse: &server.PersonTreeResponse{Members: []server.PersonWithRelationship{
		// 		{
		// 			RelationshipPerson: server.RelationshipPerson{
		// 				Name:   personInsertedIdByName["Caio"].Name,
		// 				Gender: "male",
		// 			},
		// 			Relationships: []server.Relationship{
		// 				{Person: server.RelationshipPerson{ID: personInsertedIdByName["Luis"], Name: }}
		// 			}},
		// 	}},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				if err := tt.buildFamilyTreeFunc(); err != nil {
					t.Errorf("failed to build family tree: %s", err.Error())
				}

				successRes, errorRes, statusCode, err := doGetPersonFamilyRelationshipsRequest(router, personInsertedIdByName[tt.personToSearchName])
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, tt.expectedStatusCode, statusCode)

				if tt.expectedErrorResponse != nil {
					assert.Equal(t, tt.expectedErrorResponse, errorRes)
				}

				if tt.expectedResponse != nil {
					addPersonIDToExpectedPersonTreeResponse(tt.expectedResponse.Members, personInsertedIdByName)
					assert.Equal(t, tt.expectedResponse, successRes)
				}
				teardownTest()
			}
		}(tt))
	}
}
