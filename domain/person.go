package domain

type RelationshipType string

const (
	ParentRelashionship  RelationshipType = "parent"
	ChildRelashionship   RelationshipType = "child"
	SiblingRelashionship RelationshipType = "sibling"
	NephewRelashionship  RelationshipType = "nephew"
	CousinRelashionship  RelationshipType = "cousin"
)

type GenderType string

const (
	Male   GenderType = "male"
	Female GenderType = "female"
)

func (s GenderType) IsValid() bool {
	switch s {
	case Female, Male:
		return true
	}

	return false
}

type RelationshipPerson struct {
	ID     string
	Name   string
	Gender GenderType
}

type Relationship struct {
	Person       RelationshipPerson
	Relationship RelationshipType
}

// type Person struct {
// 	Person

// 	BaconsNumber int64
// }

// Bread first search começando com Phoebe.
// Visito as crianças, visito o pai
// Vou adicionando geração quando pesquiso para o PAI e subtraindo qnd buscando pelos filhos.
// adiciono visited nodes
// crio um map com geração! e faço, geração do current node +1 -1 para pegar relationships

type FamilyTreeMember struct {
	ID         string
	Name       string
	Gender     GenderType
	Generation int

	ChildrenToVisit []*FamilyTreeMember
	ChildrenIDS     []string

	ParentToVisit []*FamilyTreeMember
	ParentIDS     []string
}

type FamilyTree struct {
	Root FamilyTreeMember
}

type Person struct {
	ID       string
	Name     string
	Gender   GenderType
	Parents  []Person
	Children []Person

	Relationships []Relationship
}

func buildRelationshipFromPerson(person Person, relationshipType RelationshipType) Relationship {
	return Relationship{
		Person: RelationshipPerson{
			ID:     person.ID,
			Name:   person.Name,
			Gender: person.Gender,
		},
		Relationship: relationshipType,
	}
}
