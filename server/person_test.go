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
	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/errors"
	"github.com/CaioBittencourt/arvore-genealogica/repository/mongodb"
	"github.com/CaioBittencourt/arvore-genealogica/server"
	"github.com/CaioBittencourt/arvore-genealogica/server/routes"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func addPersonIDToExpectedResponse(expected *server.PersonResponse, personInsertedIdByName map[string]string) {
	expected.ID = personInsertedIdByName[expected.Name]

	for i, expectedParent := range expected.Parents {
		expected.Parents[i].ID = personInsertedIdByName[expectedParent.Name]
	}

	for i, expectedChildren := range expected.Children {
		expected.Children[i].ID = personInsertedIdByName[expectedChildren.Name]
	}

	for i, expectedSpouse := range expected.Spouses {
		expected.Spouses[i].ID = personInsertedIdByName[expectedSpouse.Name]
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

// refactor this method to be generic
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

func doGetBaconsNumberBetweenTwoPersonsRequest(router *gin.Engine, personAID string, personBID string) (*server.GetBaconsNumberBetweenTwoPersonsResponse, *server.ErrorResponse, int, error) {
	var buf bytes.Buffer

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/person/%s/baconNumber/%s", personAID, personBID), &buf)
	router.ServeHTTP(w, httpReq)

	if w.Code > 200 {
		res := server.ErrorResponse{}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		if err != nil {
			return nil, nil, w.Code, err
		}

		return nil, &res, w.Code, nil
	}

	res := server.GetBaconsNumberBetweenTwoPersonsResponse{}
	err := json.Unmarshal(w.Body.Bytes(), &res)
	if err != nil {
		return nil, nil, w.Code, err
	}

	return &res, nil, w.Code, nil
}

func doGetRelationshipBetweenPersonsRequest(router *gin.Engine, personAID string, personBID string) (*server.PersonWithRelationship, *server.ErrorResponse, int, error) {
	var buf bytes.Buffer

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/person/%s/relationship/%s", personAID, personBID), &buf)
	router.ServeHTTP(w, httpReq)

	if w.Code > 200 {
		res := server.ErrorResponse{}
		err := json.Unmarshal(w.Body.Bytes(), &res)
		if err != nil {
			return nil, nil, w.Code, err
		}

		return nil, &res, w.Code, nil
	}

	res := server.PersonWithRelationship{}
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

	denise := server.StorePersonRequest{
		Name:   "Denise",
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
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: "name must have more than 1 character", ErrorCode: string(errors.InvalidPersonNameErrorCode)},
		},
		{
			testName: "should return bad request when gender is not valid",
			personToStore: server.StorePersonRequest{
				Name:   "Caio",
				Gender: "unexistingGender",
			},
			expectedStatusCode:    400,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: "gender has to be male of female", ErrorCode: string(errors.InvalidPersonGenderErrorCode)},
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
			testName:           "should store person with father",
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
			testName:           "should store person with children and identify spouse relationship through children",
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
		{
			testName:           "should store person with mother and father",
			personToStore:      denise,
			fatherName:         alfredo.Name,
			motherName:         helena.Name,
			expectedStatusCode: 200,
			expectedResponse: &server.PersonResponse{
				Name:     denise.Name,
				Gender:   denise.Gender,
				Parents:  []server.PersonRelativesResponse{{Name: helena.Name, Gender: helena.Gender}, {Name: alfredo.Name, Gender: alfredo.Gender}},
				Children: []server.PersonRelativesResponse{},
				Spouses:  []server.PersonRelativesResponse{},
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
					addPersonIDToExpectedResponse(tt.expectedResponse, personInsertedIdByName)
					assert.Equal(t, tt.expectedResponse.Name, successRes.Name)
					assert.Equal(t, tt.expectedResponse.Gender, successRes.Gender)

					assert.ElementsMatch(t, tt.expectedResponse.Parents, successRes.Parents)
					assert.ElementsMatch(t, tt.expectedResponse.Children, successRes.Children)
					assert.ElementsMatch(t, tt.expectedResponse.Spouses, successRes.Spouses)
				}
			}
		}(tt))
	}

	teardownTest()
}

func buildFamily(router *gin.Engine, personInsertedIdByName map[string]server.PersonResponse) error {

	if err := storePerson(router, server.StorePersonRequest{Name: "Tunico", Gender: "male"}, personInsertedIdByName); err != nil {
		return err
	}
	tunicoID := personInsertedIdByName["Tunico"].ID

	if err := storePerson(router, server.StorePersonRequest{Name: "Luis", Gender: "male", FatherID: &tunicoID}, personInsertedIdByName); err != nil {
		return err
	}
	luisID := personInsertedIdByName["Luis"].ID

	if err := storePerson(router, server.StorePersonRequest{Name: "Dayse", Gender: "female"}, personInsertedIdByName); err != nil {
		return err
	}
	dayseID := personInsertedIdByName["Dayse"].ID

	if err := storePerson(router, server.StorePersonRequest{Name: "Caio", Gender: "male", FatherID: &luisID, MotherID: &dayseID}, personInsertedIdByName); err != nil {
		return err
	}

	if err := storePerson(router, server.StorePersonRequest{Name: "Claudia", Gender: "female", FatherID: &tunicoID}, personInsertedIdByName); err != nil {
		return err
	}
	claudiaID := personInsertedIdByName["Claudia"].ID

	if err := storePerson(router, server.StorePersonRequest{Name: "Livia", Gender: "female", MotherID: &claudiaID}, personInsertedIdByName); err != nil {
		return err
	}

	if err := storePerson(router, server.StorePersonRequest{Name: "Cauã", Gender: "male"}, personInsertedIdByName); err != nil {
		return err
	}
	cauaID := personInsertedIdByName["Cauã"].ID

	if err := storePerson(router, server.StorePersonRequest{Name: "Vivian", Gender: "female", FatherID: &luisID, MotherID: &dayseID, ChildrenIDs: []string{cauaID}}, personInsertedIdByName); err != nil {
		return err
	}

	if err := storePerson(router, server.StorePersonRequest{Name: "Caio Regis", Gender: "male", ChildrenIDs: []string{cauaID}}, personInsertedIdByName); err != nil {
		return err
	}

	return nil
}

func storePerson(router *gin.Engine, req server.StorePersonRequest, personInsertedIdByName map[string]server.PersonResponse) error {
	resSuccess, resError, statusCode, err := doStorePersonRequest(router, req)
	if err != nil {
		return err
	}

	if statusCode != 200 || resError != nil {
		return fmt.Errorf("test: failed building family of one person. status code: %d, err: %s", statusCode, resError.ErrorMessage)
	}

	personInsertedIdByName[req.Name] = *resSuccess
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
		buildExpectedResponse func(map[string]server.PersonResponse) *server.PersonTreeResponse
		expectedErrorResponse *server.ErrorResponse
	}

	insertedPersonByName := map[string]server.PersonResponse{}

	// Unexisting ID
	insertedPersonByName["UnexistingName"] = server.PersonResponse{ID: primitive.NewObjectID().Hex()}
	tests := []testArgs{
		{
			testName: "should return 404 not found when person ID passed dont exist",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personToSearchName:    "UnexistingName",
			expectedStatusCode:    404,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: "person not found", ErrorCode: "PERSON_NOT_FOUND"},
		},
		{
			testName: "should return tree when there is just one node",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personToSearchName: "Loner",
			expectedStatusCode: 200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonTreeResponse {
				return &server.PersonTreeResponse{Members: map[string]server.PersonWithRelationship{
					insertedPersonByName["Loner"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							Name:   "Loner",
							Gender: "male",
						},
						Relationships: []server.Relationship{},
					},
				}}
			},
		},
		{
			testName: "should return tree relationships: nephew, aunt, cousin, spouse, parent, children, sibling",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personToSearchName: "Vivian",
			expectedStatusCode: 200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonTreeResponse {
				return &server.PersonTreeResponse{Members: map[string]server.PersonWithRelationship{
					insertedPersonByName["Vivian"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Vivian"].ID,
							Name:   insertedPersonByName["Vivian"].Name,
							Gender: insertedPersonByName["Vivian"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Caio Regis"].ID,
									Name:   insertedPersonByName["Caio Regis"].Name,
									Gender: insertedPersonByName["Caio Regis"].Gender,
								},
								Relationship: string(domain.SpouseRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Luis"].ID,
									Name:   insertedPersonByName["Luis"].Name,
									Gender: insertedPersonByName["Luis"].Gender,
								},
								Relationship: string(domain.ParentRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Dayse"].ID,
									Name:   insertedPersonByName["Dayse"].Name,
									Gender: insertedPersonByName["Dayse"].Gender,
								},
								Relationship: string(domain.ParentRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Cauã"].ID,
									Name:   insertedPersonByName["Cauã"].Name,
									Gender: insertedPersonByName["Cauã"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Livia"].ID,
									Name:   insertedPersonByName["Livia"].Name,
									Gender: insertedPersonByName["Livia"].Gender,
								},
								Relationship: string(domain.CousinRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Caio"].ID,
									Name:   insertedPersonByName["Caio"].Name,
									Gender: insertedPersonByName["Caio"].Gender,
								},
								Relationship: string(domain.SiblingRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Claudia"].ID,
									Name:   insertedPersonByName["Claudia"].Name,
									Gender: insertedPersonByName["Claudia"].Gender,
								},
								Relationship: string(domain.AuntUncleRelashionship),
							},
						},
					},
					insertedPersonByName["Tunico"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Tunico"].ID,
							Name:   insertedPersonByName["Tunico"].Name,
							Gender: insertedPersonByName["Tunico"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Claudia"].ID,
									Name:   insertedPersonByName["Claudia"].Name,
									Gender: insertedPersonByName["Claudia"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
						},
					},
					insertedPersonByName["Dayse"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Dayse"].ID,
							Name:   insertedPersonByName["Dayse"].Name,
							Gender: insertedPersonByName["Dayse"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Caio"].ID,
									Name:   insertedPersonByName["Caio"].Name,
									Gender: insertedPersonByName["Caio"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
						},
					},
					insertedPersonByName["Claudia"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Claudia"].ID,
							Name:   insertedPersonByName["Claudia"].Name,
							Gender: insertedPersonByName["Claudia"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Livia"].ID,
									Name:   insertedPersonByName["Livia"].Name,
									Gender: insertedPersonByName["Livia"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
						},
					},
					insertedPersonByName["Luis"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Luis"].ID,
							Name:   insertedPersonByName["Luis"].Name,
							Gender: insertedPersonByName["Luis"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Tunico"].ID,
									Name:   insertedPersonByName["Tunico"].Name,
									Gender: insertedPersonByName["Tunico"].Gender,
								},
								Relationship: string(domain.ParentRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Dayse"].ID,
									Name:   insertedPersonByName["Dayse"].Name,
									Gender: insertedPersonByName["Dayse"].Gender,
								},
								Relationship: string(domain.SpouseRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Livia"].ID,
									Name:   insertedPersonByName["Livia"].Name,
									Gender: insertedPersonByName["Livia"].Gender,
								},
								Relationship: string(domain.NephewRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Claudia"].ID,
									Name:   insertedPersonByName["Claudia"].Name,
									Gender: insertedPersonByName["Claudia"].Gender,
								},
								Relationship: string(domain.SiblingRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Caio"].ID,
									Name:   insertedPersonByName["Caio"].Name,
									Gender: insertedPersonByName["Caio"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
						},
					},
					insertedPersonByName["Livia"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Livia"].ID,
							Name:   insertedPersonByName["Livia"].Name,
							Gender: insertedPersonByName["Livia"].Gender,
						},
						Relationships: []server.Relationship{},
					},
					insertedPersonByName["Caio"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Caio"].ID,
							Name:   insertedPersonByName["Caio"].Name,
							Gender: insertedPersonByName["Caio"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Livia"].ID,
									Name:   insertedPersonByName["Livia"].Name,
									Gender: insertedPersonByName["Livia"].Gender,
								},
								Relationship: string(domain.CousinRelashionship),
							},
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Claudia"].ID,
									Name:   insertedPersonByName["Claudia"].Name,
									Gender: insertedPersonByName["Claudia"].Gender,
								},
								Relationship: string(domain.AuntUncleRelashionship),
							},
						},
					},
					insertedPersonByName["Caio Regis"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Caio Regis"].ID,
							Name:   insertedPersonByName["Caio Regis"].Name,
							Gender: insertedPersonByName["Caio Regis"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Cauã"].ID,
									Name:   insertedPersonByName["Cauã"].Name,
									Gender: insertedPersonByName["Cauã"].Gender,
								},
								Relationship: string(domain.ChildRelashionship),
							},
						},
					},
					insertedPersonByName["Cauã"].ID: {
						RelationshipPerson: server.RelationshipPerson{
							ID:     insertedPersonByName["Cauã"].ID,
							Name:   insertedPersonByName["Cauã"].Name,
							Gender: insertedPersonByName["Cauã"].Gender,
						},
						Relationships: []server.Relationship{
							{
								Person: server.RelationshipPerson{
									ID:     insertedPersonByName["Caio"].ID,
									Name:   insertedPersonByName["Caio"].Name,
									Gender: insertedPersonByName["Caio"].Gender,
								},
								Relationship: string(domain.AuntUncleRelashionship),
							},
						},
					},
				}}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				if err := tt.buildFamilyTreeFunc(); err != nil {
					t.Errorf("failed to build family tree: %s", err.Error())
				}

				successRes, errorRes, statusCode, err := doGetPersonFamilyRelationshipsRequest(router, insertedPersonByName[tt.personToSearchName].ID)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, tt.expectedStatusCode, statusCode)

				if tt.expectedErrorResponse != nil {
					assert.Equal(t, tt.expectedErrorResponse, errorRes)
					return
				}

				expectedResponse := tt.buildExpectedResponse(insertedPersonByName)

				if expectedResponse != nil {
					for _, personWithRelationships := range expectedResponse.Members {
						assert.ElementsMatch(
							t,
							personWithRelationships.Relationships,
							successRes.Members[personWithRelationships.ID].Relationships,
						)
					}
				}
				teardownTest()
			}
		}(tt))
	}
}

func TestGetBaconsNumberBetweenTwoPersons(t *testing.T) {
	//NOTE: Leaving this settup per test in case any tests want to introduce a mock for controllers or repository.
	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personController := controller.NewPersonController(personRepository)
	router := routes.SetupRouter(personController)

	type testArgs struct {
		testName              string
		buildFamilyTreeFunc   func() error
		personAToSearchName   string
		personBToSearchName   string
		expectedStatusCode    int
		expectedResponse      *server.GetBaconsNumberBetweenTwoPersonsResponse
		expectedErrorResponse *server.ErrorResponse
	}

	insertedPersonByName := map[string]server.PersonResponse{}

	// Unexisting ID
	insertedPersonByName["UnexistingNameA"] = server.PersonResponse{ID: primitive.NewObjectID().Hex()}
	insertedPersonByName["UnexistingNameB"] = server.PersonResponse{ID: primitive.NewObjectID().Hex()}
	tests := []testArgs{
		{
			testName: "should return 404 not found when person ID A passed dont exist",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName:   "UnexistingNameA",
			personBToSearchName:   "Loner",
			expectedStatusCode:    404,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: fmt.Sprintf("person with id %s not found", insertedPersonByName["UnexistingNameA"].ID), ErrorCode: "PERSON_NOT_FOUND"},
		},
		{
			testName: "should return 404 not found when person ID B passed dont exist",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName:   "Loner",
			personBToSearchName:   "UnexistingNameB",
			expectedStatusCode:    404,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: fmt.Sprintf("person with id %s not found", insertedPersonByName["UnexistingNameB"].ID), ErrorCode: "PERSON_NOT_FOUND"},
		},
		{
			testName: "should return bacons number for grandparents",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Tunico",
			personBToSearchName: "Caio",
			expectedStatusCode:  200,
			expectedResponse: &server.GetBaconsNumberBetweenTwoPersonsResponse{
				BaconsNumber: 2,
			},
		},
		{
			testName: "should return bacons number for cousins",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Livia",
			personBToSearchName: "Caio",
			expectedStatusCode:  200,
			expectedResponse: &server.GetBaconsNumberBetweenTwoPersonsResponse{
				BaconsNumber: 4,
			},
		},
		{
			testName: "should return bacons number for spouses and should be from graph relationship spouse",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Luis",
			personBToSearchName: "Dayse",
			expectedStatusCode:  200,
			expectedResponse: &server.GetBaconsNumberBetweenTwoPersonsResponse{
				BaconsNumber: 1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				if err := tt.buildFamilyTreeFunc(); err != nil {
					t.Errorf("failed to build family tree: %s", err.Error())
				}

				successRes, errorRes, statusCode, err := doGetBaconsNumberBetweenTwoPersonsRequest(
					router,
					insertedPersonByName[tt.personAToSearchName].ID,
					insertedPersonByName[tt.personBToSearchName].ID,
				)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, tt.expectedStatusCode, statusCode)

				if tt.expectedErrorResponse != nil {
					assert.Equal(t, tt.expectedErrorResponse, errorRes)
					return
				}

				if tt.expectedResponse != nil {
					assert.Equal(t, tt.expectedResponse, successRes)
				}
				teardownTest()
			}
		}(tt))
	}
}

func TestGetPersonFamilyRelationships(t *testing.T) {
	//NOTE: Leaving this settup per test in case any tests want to introduce a mock for controllers or repository.
	personRepository := mongodb.NewPersonRepository(*mongoClient, os.Getenv("MONGO_DATABASE"))
	personController := controller.NewPersonController(personRepository)
	router := routes.SetupRouter(personController)

	type testArgs struct {
		testName              string
		buildFamilyTreeFunc   func() error
		personAToSearchName   string
		personBToSearchName   string
		expectedStatusCode    int
		buildExpectedResponse func(map[string]server.PersonResponse) *server.PersonWithRelationship
		expectedErrorResponse *server.ErrorResponse
	}

	insertedPersonByName := map[string]server.PersonResponse{}

	// Unexisting ID
	insertedPersonByName["UnexistingNameA"] = server.PersonResponse{ID: primitive.NewObjectID().Hex()}
	insertedPersonByName["UnexistingNameB"] = server.PersonResponse{ID: primitive.NewObjectID().Hex()}
	tests := []testArgs{
		{
			testName: "should return 404 not found when person ID A passed dont exist",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName:   "UnexistingNameA",
			personBToSearchName:   "Loner",
			expectedStatusCode:    404,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: fmt.Sprintf("person with id %s not found", insertedPersonByName["UnexistingNameA"].ID), ErrorCode: "PERSON_NOT_FOUND"},
		},
		{
			testName: "should return 404 not found when person ID B passed dont exist",
			buildFamilyTreeFunc: func() error {
				if err := storePerson(router, server.StorePersonRequest{Name: "Loner", Gender: "male"}, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName:   "Loner",
			personBToSearchName:   "UnexistingNameB",
			expectedStatusCode:    404,
			expectedErrorResponse: &server.ErrorResponse{ErrorMessage: fmt.Sprintf("person with id %s not found", insertedPersonByName["UnexistingNameB"].ID), ErrorCode: "PERSON_NOT_FOUND"},
		},
		{
			testName: "should return cousin relationship",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Caio",
			personBToSearchName: "Livia",
			expectedStatusCode:  200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonWithRelationship {
				return &server.PersonWithRelationship{
					RelationshipPerson: server.RelationshipPerson{
						Name:   insertedPersonByName["Caio"].Name,
						ID:     insertedPersonByName["Caio"].ID,
						Gender: insertedPersonByName["Caio"].Gender,
					},
					Relationships: []server.Relationship{
						{
							Person: server.RelationshipPerson{
								Name:   insertedPersonByName["Livia"].Name,
								ID:     insertedPersonByName["Livia"].ID,
								Gender: insertedPersonByName["Livia"].Gender,
							},
							Relationship: string(domain.CousinRelashionship),
						},
					},
				}
			},
		},
		{
			testName: "should return spouse relationship",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Dayse",
			personBToSearchName: "Luis",
			expectedStatusCode:  200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonWithRelationship {
				return &server.PersonWithRelationship{
					RelationshipPerson: server.RelationshipPerson{
						Name:   insertedPersonByName["Dayse"].Name,
						ID:     insertedPersonByName["Dayse"].ID,
						Gender: insertedPersonByName["Dayse"].Gender,
					},
					Relationships: []server.Relationship{
						{
							Person: server.RelationshipPerson{
								Name:   insertedPersonByName["Luis"].Name,
								ID:     insertedPersonByName["Luis"].ID,
								Gender: insertedPersonByName["Luis"].Gender,
							},
							Relationship: string(domain.SpouseRelashionship),
						},
					},
				}
			},
		},
		{
			testName: "should return nephew relationship",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Caio",
			personBToSearchName: "Cauã",
			expectedStatusCode:  200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonWithRelationship {
				return &server.PersonWithRelationship{
					RelationshipPerson: server.RelationshipPerson{
						Name:   insertedPersonByName["Caio"].Name,
						ID:     insertedPersonByName["Caio"].ID,
						Gender: insertedPersonByName["Caio"].Gender,
					},
					Relationships: []server.Relationship{
						{
							Person: server.RelationshipPerson{
								Name:   insertedPersonByName["Cauã"].Name,
								ID:     insertedPersonByName["Cauã"].ID,
								Gender: insertedPersonByName["Cauã"].Gender,
							},
							Relationship: string(domain.NephewRelashionship),
						},
					},
				}
			},
		},
		{
			testName: "should return sibling relationship",
			buildFamilyTreeFunc: func() error {
				if err := buildFamily(router, insertedPersonByName); err != nil {
					return err
				}
				return nil
			},
			personAToSearchName: "Vivian",
			personBToSearchName: "Caio",
			expectedStatusCode:  200,
			buildExpectedResponse: func(insertedPersonByName map[string]server.PersonResponse) *server.PersonWithRelationship {
				return &server.PersonWithRelationship{
					RelationshipPerson: server.RelationshipPerson{
						Name:   insertedPersonByName["Vivian"].Name,
						ID:     insertedPersonByName["Vivian"].ID,
						Gender: insertedPersonByName["Vivian"].Gender,
					},
					Relationships: []server.Relationship{
						{
							Person: server.RelationshipPerson{
								Name:   insertedPersonByName["Caio"].Name,
								ID:     insertedPersonByName["Caio"].ID,
								Gender: insertedPersonByName["Caio"].Gender,
							},
							Relationship: string(domain.SiblingRelashionship),
						},
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				if err := tt.buildFamilyTreeFunc(); err != nil {
					t.Errorf("failed to build family tree: %s", err.Error())
				}

				successRes, errorRes, statusCode, err := doGetRelationshipBetweenPersonsRequest(
					router,
					insertedPersonByName[tt.personAToSearchName].ID,
					insertedPersonByName[tt.personBToSearchName].ID,
				)
				if err != nil {
					t.Error(err)
				}

				assert.Equal(t, tt.expectedStatusCode, statusCode)

				if tt.expectedErrorResponse != nil {
					assert.Equal(t, tt.expectedErrorResponse, errorRes)
					return
				}

				expectedResponse := tt.buildExpectedResponse(insertedPersonByName)
				if expectedResponse != nil {
					assert.Equal(t, expectedResponse, successRes)
				}
				teardownTest()
			}
		}(tt))
	}
}
