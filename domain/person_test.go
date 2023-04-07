package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func buildFamilyGraph() FamilyGraph {
	livia := &Person{ID: "IDLivia", Name: "Livia", Gender: Female, Generation: 0}
	claudia := &Person{ID: "IDClaudia", Name: "Claudia", Gender: Female, Generation: 1}
	zeze := &Person{ID: "IDZézé", Name: "Zézé", Gender: Female, Generation: 2}
	luis := &Person{ID: "IDLuis", Name: "Luis", Gender: Male, Generation: 1}
	dayse := &Person{ID: "IDDayse", Name: "Dayse", Gender: Female, Generation: 1}
	vivian := &Person{ID: "IDVivian", Name: "Vivian", Gender: Female, Generation: 0}
	caio := &Person{ID: "IDCaio", Name: "Caio", Gender: Male, Generation: 0}
	caua := &Person{ID: "IDCauã", Name: "Cauã", Gender: Male, Generation: -1}

	livia.Parents = []*Person{claudia}

	claudia.Children = []*Person{livia}
	claudia.Parents = []*Person{zeze}

	zeze.Children = []*Person{luis, claudia}

	luis.Children = []*Person{caio, vivian}
	luis.Parents = []*Person{zeze}
	luis.Spouses = []*Person{dayse}

	dayse.Spouses = []*Person{luis}
	dayse.Children = []*Person{caio, vivian}

	caio.Parents = []*Person{dayse, luis}

	vivian.Parents = []*Person{dayse, luis}
	vivian.Children = []*Person{caua}

	caua.Parents = []*Person{vivian}

	return FamilyGraph{
		Members: map[string]*Person{
			caio.ID:    caio,
			livia.ID:   livia,
			claudia.ID: claudia,
			zeze.ID:    zeze,
			luis.ID:    luis,
			dayse.ID:   dayse,
			vivian.ID:  vivian,
			caua.ID:    caua,
		}}
}

func TestBuildFamilyRelationships(t *testing.T) {
	type testArgs struct {
		testName                      string
		familyGraph                   FamilyGraph
		searchedPersonID              string
		expectedRelationshipsByPerson map[string][]Relationship
		errorMessage                  string
	}

	familyGraph := buildFamilyGraph()
	tests := []testArgs{
		{
			testName:         "should error because person was not found in graph",
			familyGraph:      familyGraph,
			searchedPersonID: "unexistingID",
			errorMessage:     "person not in graph",
		},
		{
			testName:         "should build relationships sucessfully",
			familyGraph:      familyGraph,
			searchedPersonID: "IDCaio",
			expectedRelationshipsByPerson: map[string][]Relationship{
				"IDCaio": []Relationship{
					{Person: RelationshipPerson{ID: "IDLuis"}, Relationship: ParentRelashionship},
					{Person: RelationshipPerson{ID: "IDDayse"}, Relationship: ParentRelashionship},
					{Person: RelationshipPerson{ID: "IDVivian"}, Relationship: SiblingRelashionship},
					{Person: RelationshipPerson{ID: "IDClaudia"}, Relationship: AuntUncleRelashionship},
					{Person: RelationshipPerson{ID: "IDCauã"}, Relationship: NephewRelashionship},
					{Person: RelationshipPerson{ID: "IDLivia"}, Relationship: CousinRelashionship},
				},
				"IDLuis": []Relationship{
					{Person: RelationshipPerson{ID: "IDZézé"}, Relationship: ParentRelashionship},
					{Person: RelationshipPerson{ID: "IDVivian"}, Relationship: ChildRelashionship},
					{Person: RelationshipPerson{ID: "IDClaudia"}, Relationship: SiblingRelashionship},
					{Person: RelationshipPerson{ID: "IDLivia"}, Relationship: NephewRelashionship},
				},
				"IDDayse": []Relationship{
					{Person: RelationshipPerson{ID: "IDLuis"}, Relationship: SpouseRelashionship},
					{Person: RelationshipPerson{ID: "IDVivian"}, Relationship: ChildRelashionship},
				},
				"IDZézé": []Relationship{
					{Person: RelationshipPerson{ID: "IDClaudia"}, Relationship: ChildRelashionship},
				},
				"IDClaudia": []Relationship{
					{Person: RelationshipPerson{ID: "IDLivia"}, Relationship: ChildRelashionship},
				},
				"IDVivian": []Relationship{
					{Person: RelationshipPerson{ID: "IDCauã"}, Relationship: ChildRelashionship},
					{Person: RelationshipPerson{ID: "IDClaudia"}, Relationship: AuntUncleRelashionship},
					{Person: RelationshipPerson{ID: "IDLivia"}, Relationship: CousinRelashionship},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				if err := tt.familyGraph.PopulateFamilyWithRelationships(tt.searchedPersonID); err != nil {
					if tt.errorMessage != "" {
						assert.Equal(t, tt.errorMessage, err.Error())
						return
					} else {
						t.Errorf("unexpected error occurred: %s", err.Error())
					}
				}

				for personID, personWithRelationships := range tt.expectedRelationshipsByPerson {
					for _, expectedRelationship := range personWithRelationships {
						assert.Equal(
							t,
							expectedRelationship.Relationship,
							tt.familyGraph.Members[personID].Relationships[expectedRelationship.Person.ID].Relationship,
						)
					}
				}
			}
		}(tt))
	}
}

func TestBaconsNumber(t *testing.T) {
	type baconsNumberArgs struct {
		personIDA string
		personIDB string
	}

	type testArgs struct {
		testName             string
		familyGraph          FamilyGraph
		baconsNumberArgs     baconsNumberArgs
		expectedBaconsNumber *uint
	}

	familyGraph := buildFamilyGraph()
	tests := []testArgs{
		{
			testName:             "should return nil bacon number if person A is not found on the family graph",
			familyGraph:          familyGraph,
			baconsNumberArgs:     baconsNumberArgs{personIDA: "unexistingID", personIDB: "IDLuis"},
			expectedBaconsNumber: nil,
		},
		{
			testName:             "should return nil bacon number if person B is not found on the family graph",
			familyGraph:          familyGraph,
			baconsNumberArgs:     baconsNumberArgs{personIDA: "IDLuis", personIDB: "unexistingID"},
			expectedBaconsNumber: nil,
		},
		{
			testName:         "should return correct bacons number between uncle / nephew",
			familyGraph:      familyGraph,
			baconsNumberArgs: baconsNumberArgs{personIDA: "IDLuis", personIDB: "IDLivia"},
			expectedBaconsNumber: func() *uint {
				baconsNumber := uint(3)
				return &baconsNumber
			}(),
		},
		{
			testName:         "should return correct bacons number for grandparents",
			familyGraph:      familyGraph,
			baconsNumberArgs: baconsNumberArgs{personIDA: "IDCaio", personIDB: "IDZézé"},
			expectedBaconsNumber: func() *uint {
				baconsNumber := uint(2)
				return &baconsNumber
			}(),
		},
		{
			testName:         "should return correct bacons number for cousins",
			familyGraph:      familyGraph,
			baconsNumberArgs: baconsNumberArgs{personIDA: "IDCaio", personIDB: "IDLivia"},
			expectedBaconsNumber: func() *uint {
				baconsNumber := uint(4)
				return &baconsNumber
			}(),
		},
		{
			testName:         "should return correct bacons number for spouses",
			familyGraph:      familyGraph,
			baconsNumberArgs: baconsNumberArgs{personIDA: "IDDayse", personIDB: "IDLuis"},
			expectedBaconsNumber: func() *uint {
				baconsNumber := uint(1)
				return &baconsNumber
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				baconsNumber := tt.familyGraph.BaconsNumber(tt.baconsNumberArgs.personIDA, tt.baconsNumberArgs.personIDB)
				if tt.expectedBaconsNumber == nil {
					assert.Equal(t, tt.expectedBaconsNumber, baconsNumber)
				} else {
					assert.Equal(t, *tt.expectedBaconsNumber, *baconsNumber)
				}
			}
		}(tt))
	}
}

func TestFindRelationshipBetweenPersons(t *testing.T) {
	type findRelationshipArgs struct {
		personIDA string
		personIDB string
	}

	type testArgs struct {
		testName             string
		familyGraph          FamilyGraph
		findRelationshipArgs findRelationshipArgs
		expectedRelationship *Person
	}

	familyGraph := buildFamilyGraph()
	tests := []testArgs{
		{
			testName:             "should return nil person with relationship if person A is not found on the family graph",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "unexistingID", personIDB: "IDLuis"},
			expectedRelationship: nil,
		},
		{
			testName:             "should return nil person with relationship if person B is not found on the family graph",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDLuis", personIDB: "unexistingID"},
			expectedRelationship: nil,
		},
		{
			testName:             "should return correct person with relationship between cousins",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDCaio", personIDB: "IDLivia"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDCaio"].ID,
				Relationships: map[string]Relationship{
					"IDLivia": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDLivia"].ID,
						},
						Relationship: CousinRelashionship,
					},
				},
			},
		},
		{
			testName:             "should return correct bacons number between spouse",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDLuis", personIDB: "IDDayse"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDLuis"].ID,
				Relationships: map[string]Relationship{
					"IDDayse": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDDayse"].ID,
						},
						Relationship: SpouseRelashionship,
					},
				},
			},
		},
		{
			testName:             "should return correct person with relationship between spouse",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDLuis", personIDB: "IDDayse"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDLuis"].ID,
				Relationships: map[string]Relationship{
					"IDDayse": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDDayse"].ID,
						},
						Relationship: SpouseRelashionship,
					},
				},
			},
		},
		{
			testName:             "should return correct person with relationship for nephew",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDCaio", personIDB: "IDCauã"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDCaio"].ID,
				Relationships: map[string]Relationship{
					"IDCauã": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDCauã"].ID,
						},
						Relationship: NephewRelashionship,
					},
				},
			},
		},
		{
			testName:             "should return correct person with relationship for siblings",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDCaio", personIDB: "IDVivian"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDCaio"].ID,
				Relationships: map[string]Relationship{
					"IDVivian": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDVivian"].ID,
						},
						Relationship: SiblingRelashionship,
					},
				},
			},
		},
		{
			testName:             "should return correct person with relationship for aunt / uncle",
			familyGraph:          familyGraph,
			findRelationshipArgs: findRelationshipArgs{personIDA: "IDVivian", personIDB: "IDClaudia"},
			expectedRelationship: &Person{
				ID: familyGraph.Members["IDVivian"].ID,
				Relationships: map[string]Relationship{
					"IDClaudia": Relationship{
						Person: RelationshipPerson{
							ID: familyGraph.Members["IDClaudia"].ID,
						},
						Relationship: AuntUncleRelashionship,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(tt testArgs) func(t *testing.T) {
			return func(t *testing.T) {
				personWithRelationship := tt.familyGraph.FindRelationshipBetweenPersons(tt.findRelationshipArgs.personIDA, tt.findRelationshipArgs.personIDB)
				if tt.expectedRelationship == nil {
					assert.Equal(t, tt.expectedRelationship, personWithRelationship)
				} else {
					assert.Equal(t, tt.expectedRelationship.ID, personWithRelationship.ID)
					for personID, relationship := range tt.expectedRelationship.Relationships {
						assert.Equal(t, relationship.Person.ID, personWithRelationship.Relationships[personID].Person.ID)
					}
				}
			}
		}(tt))
	}
}
