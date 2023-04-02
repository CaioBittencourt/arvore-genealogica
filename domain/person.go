package domain

import (
	"container/list"
	"errors"
	"fmt"
)

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

// type FamilyTreeMember struct {
// 	ID         string
// 	Name       string
// 	Gender     GenderType
// 	Generation int

// 	ChildrenToVisit []*FamilyTreeMember
// 	ChildrenIDS     []string

// 	ParentToVisit []*FamilyTreeMember
// 	ParentIDS     []string
// }

type FamilyTree struct {
	Root Person
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
		fmt.Println(currentPerson.Name, currentPerson.ID)

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

		previousGeneration := currentPerson.Generation - 1
		personsToCheckRelationship, ok := personByGeneration[previousGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				// check for children / nephew / cousins
				if currentPerson.isChildren(*currentPersonToCheck) {
					currentPerson.Relationships = append(currentPerson.Relationships, buildRelationshipWithPerson(*currentPersonToCheck, ChildRelashionship))
					continue
				}

				if currentPerson.isNephew(*currentPersonToCheck) {
					currentPerson.Relationships = append(currentPerson.Relationships, buildRelationshipWithPerson(*currentPersonToCheck, NephewRelashionship))
					continue
				}

				if currentPerson.isCousin(*currentPersonToCheck) {
					currentPerson.Relationships = append(currentPerson.Relationships, buildRelationshipWithPerson(*currentPersonToCheck, CousinRelashionship))
					continue
				}
			}
		}

		nextGeneration := currentPerson.Generation + 1
		personsToCheckRelationship, ok = personByGeneration[nextGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				if currentPerson.isParent(*currentPersonToCheck) {
					currentPerson.Relationships = append(currentPerson.Relationships, buildRelationshipWithPerson(*currentPersonToCheck, ParentRelashionship))
					continue
				}
			}
		}

		currentGeneration := currentPerson.Generation
		personsToCheckRelationship, ok = personByGeneration[currentGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				if currentPerson.isSibling(*currentPersonToCheck) {
					currentPerson.Relationships = append(currentPerson.Relationships, buildRelationshipWithPerson(*currentPersonToCheck, SiblingRelashionship))
					continue
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
