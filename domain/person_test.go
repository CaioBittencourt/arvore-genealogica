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
				if err := tt.familyGraph.PopulateWithFamilyRelationships(tt.searchedPersonID); err != nil {
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
