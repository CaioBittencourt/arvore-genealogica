package domain

import (
	"container/list"
	"errors"
)

type RelationshipType string

const (
	ParentRelashionship    RelationshipType = "parent"
	ChildRelashionship     RelationshipType = "child"
	SiblingRelashionship   RelationshipType = "sibling"
	NephewRelashionship    RelationshipType = "nephew"
	CousinRelashionship    RelationshipType = "cousin"
	SpouseRelashionship    RelationshipType = "spouse"
	AuntUncleRelashionship RelationshipType = "aunt/uncle"
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
type FamilyGraph struct {
	Members map[string]*Person
}
type Person struct {
	ID            string
	Name          string
	Gender        GenderType
	Parents       []*Person
	Children      []*Person
	Generation    int
	Relationships []Relationship
}

func buildRelationshipWithPerson(person Person, relationshipType RelationshipType) Relationship {
	return Relationship{
		Person: RelationshipPerson{
			ID:     person.ID,
			Name:   person.Name,
			Gender: person.Gender,
		},
		Relationship: relationshipType,
	}
}

func (p Person) isParent(possibleParent Person) bool {
	for _, parent := range p.Parents {
		if parent.ID == possibleParent.ID {
			return true
		}
	}

	return false
}

func (p Person) isChildren(possibleChildren Person) bool {
	for _, children := range p.Children {
		if children.ID == possibleChildren.ID {
			return true
		}
	}

	return false
}

func (p Person) isNephew(possibleNephew Person) bool {
	for _, parent := range p.Parents {
		for _, parentChildren := range parent.Children {
			for _, possibleNephewParent := range possibleNephew.Parents {
				if possibleNephewParent.ID == parentChildren.ID {
					return true
				}
			}
		}
	}

	return false
}

func (p Person) isSibling(possibleSibling Person) bool {
	if possibleSibling.ID == p.ID {
		return false
	}

	for _, parent := range p.Parents {
		for _, possibleSiblingParent := range possibleSibling.Parents {
			if possibleSiblingParent.ID == parent.ID {
				return true
			}
		}
	}

	return false
}

func (p Person) isCousin(possibleCousin Person) bool {
	for _, parent := range p.Parents {
		for _, grandparent := range parent.Parents {
			for _, grandparentChildren := range grandparent.Children {
				for _, possibleCousinParent := range possibleCousin.Parents {
					if possibleCousinParent.ID == grandparentChildren.ID {
						return true
					}
				}
			}
		}
	}

	return false
}

func (p Person) isSpouse(possibleSpouse Person) bool {
	for _, children := range p.Children {
		for _, possibleSpouseChildren := range possibleSpouse.Children {
			if possibleSpouseChildren.ID == children.ID {
				return true
			}
		}
	}

	return false
}

func (p Person) isUncleOrAunt(possibleUncle Person) bool {
	for _, parent := range p.Parents {
		for _, grandParent := range parent.Parents {
			for _, possibleUncle := range possibleUncle.Parents {
				if possibleUncle.ID == grandParent.ID {
					return true
				}
			}
		}
	}

	return false
}

func (p Person) FindNextGenerationRelationships(personToFindRelationships Person) *Relationship {
	if p.isParent(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, ParentRelashionship)
		return &relationship
	}

	if p.isUncleOrAunt(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, AuntUncleRelashionship)
		return &relationship
	}

	return nil
}

func (p Person) FindPreviouseGenerationRelationships(personToFindRelationships Person) *Relationship {
	if p.isChildren(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, ChildRelashionship)
		return &relationship
	}

	if p.isNephew(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, NephewRelashionship)
		return &relationship
	}

	if p.isCousin(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, CousinRelashionship)
		return &relationship
	}

	return nil
}

func (p Person) FindCurrentGenerationRelationships(personToFindRelationships Person) *Relationship {
	if p.isSibling(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, SiblingRelashionship)
		return &relationship
	}

	if p.isSpouse(personToFindRelationships) {
		relationship := buildRelationshipWithPerson(personToFindRelationships, SpouseRelashionship)
		return &relationship
	}

	return nil
}

func (fg *FamilyGraph) BuildFamilyRelationships(personID string) error {
	currentPerson, ok := fg.Members[personID]
	if !ok {
		return errors.New("person not in graph")
	}

	visitPersonQueue := list.New()
	personAlreadyOnQueue := make(map[string]bool)
	personAlreadyOnQueue[currentPerson.ID] = true

	personByGeneration := make(map[int][]*Person)
	for currentPerson != nil || visitPersonQueue.Len() > 0 {
		for _, currentParent := range currentPerson.Parents {
			if _, ok := personAlreadyOnQueue[currentParent.ID]; !ok {
				visitPersonQueue.PushBack(currentParent)
				personAlreadyOnQueue[currentParent.ID] = true
			}
		}

		for _, currentChildren := range currentPerson.Children {
			if _, ok := personAlreadyOnQueue[currentChildren.ID]; !ok {
				visitPersonQueue.PushBack(currentChildren)
				personAlreadyOnQueue[currentChildren.ID] = true
			}
		}

		nextGeneration := currentPerson.Generation + 1
		personsToCheckRelationship, ok := personByGeneration[nextGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				relationship := currentPerson.FindNextGenerationRelationships(*currentPersonToCheck)
				if relationship != nil {
					currentPerson.Relationships = append(currentPerson.Relationships, *relationship)
				}
			}
		}

		currentGeneration := currentPerson.Generation
		personsToCheckRelationship, ok = personByGeneration[currentGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				relationship := currentPerson.FindCurrentGenerationRelationships(*currentPersonToCheck)
				if relationship != nil {
					currentPerson.Relationships = append(currentPerson.Relationships, *relationship)
				}
			}
		}

		previousGeneration := currentPerson.Generation - 1
		personsToCheckRelationship, ok = personByGeneration[previousGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				relationship := currentPerson.FindPreviouseGenerationRelationships(*currentPersonToCheck)
				if relationship != nil {
					currentPerson.Relationships = append(currentPerson.Relationships, *relationship)
				}
			}
		}

		personByGeneration[currentPerson.Generation] = append(personByGeneration[currentPerson.Generation], currentPerson)

		nextPerson := visitPersonQueue.Front()
		if nextPerson != nil {
			visitPersonQueue.Remove(nextPerson)
			person := nextPerson.Value.(*Person)
			currentPerson = person
		} else {
			currentPerson = nil
		}
	}

	return nil
}
