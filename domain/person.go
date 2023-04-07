package domain

import (
	"container/list"
	"math"

	"github.com/CaioBittencourt/arvore-genealogica/errors"
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
	ID       string
	Name     string
	Gender   GenderType
	Parents  []*Person
	Children []*Person

	Spouses       []*Person
	Generation    int
	Relationships map[string]Relationship
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

func (p Person) Validate() error {
	if len(p.Parents) > 2 {
		return errors.NewApplicationError("not allowed to have more than 2 parents", errors.TooManyParentsForPersonErrorCode)
	}

	if len(p.Name) < 2 {
		return errors.NewApplicationError("name must have more than 1 character", errors.InvalidPersonNameErrorCode)
	}

	if !p.Gender.IsValid() {
		return errors.NewApplicationError("gender has to be male of female", errors.InvalidPersonGenderErrorCode)
	}

	return nil
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

func (p Person) FindNextGenerationRelationships(personToFindRelationship Person) *Relationship {
	if p.isParent(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, ParentRelashionship)
		return &relationship
	}

	if p.isUncleOrAunt(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, AuntUncleRelashionship)
		return &relationship
	}

	return nil
}

func (p Person) FindPreviouseGenerationRelationships(personToFindRelationship Person) *Relationship {
	if p.isChildren(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, ChildRelashionship)
		return &relationship
	}

	if p.isNephew(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, NephewRelashionship)
		return &relationship
	}

	return nil
}

func (p Person) FindCurrentGenerationRelationships(personToFindRelationship Person) *Relationship {
	if p.isSibling(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, SiblingRelashionship)
		return &relationship
	}

	if p.isSpouse(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, SpouseRelashionship)
		return &relationship
	}

	if p.isCousin(personToFindRelationship) {
		relationship := buildRelationshipWithPerson(personToFindRelationship, CousinRelashionship)
		return &relationship
	}

	return nil
}

func (fg FamilyGraph) FindRelationshipBetweenPersons(personAID string, personBID string) *Person {
	personA, ok := fg.Members[personAID]
	if !ok {
		return nil
	}

	personB, ok := fg.Members[personBID]
	if !ok {
		return nil
	}

	generationToSearch := personA.Generation - personB.Generation
	generationDiff := math.Abs(float64(generationToSearch))
	if generationDiff > 1 {
		return nil
	}

	personA.Relationships = map[string]Relationship{}
	if generationToSearch == -1 {
		relationship := personA.FindNextGenerationRelationships(*personB)
		if relationship != nil {
			personA.Relationships[personB.ID] = *relationship
		}
	} else if generationToSearch == 0 {
		relationship := personA.FindCurrentGenerationRelationships(*personB)
		if relationship != nil {
			personA.Relationships[personB.ID] = *relationship
		}
	} else if generationToSearch == 1 {
		relationship := personA.FindPreviouseGenerationRelationships(*personB)
		if relationship != nil {
			personA.Relationships[personB.ID] = *relationship
		}
	}

	return personA

}

func (fg *FamilyGraph) PopulateFamilyWithRelationships(personID string) error {
	currentPerson, ok := fg.Members[personID]
	if !ok {
		return errors.NewApplicationError("person not in graph", errors.PersonNotFoundInGraph)
	}

	visitPersonQueue := list.New()
	personAlreadyOnQueue := make(map[string]bool)
	personAlreadyOnQueue[currentPerson.ID] = true

	personByGeneration := make(map[int][]*Person)
	for currentPerson != nil || visitPersonQueue.Len() > 0 {
		currentPerson.Relationships = make(map[string]Relationship)

		for _, currentSpouse := range currentPerson.Spouses {
			if _, ok := personAlreadyOnQueue[currentSpouse.ID]; !ok {
				visitPersonQueue.PushBack(currentSpouse)
				personAlreadyOnQueue[currentSpouse.ID] = true
			}
		}

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
				relationship := currentPersonToCheck.FindPreviouseGenerationRelationships(*currentPerson)
				if relationship != nil {
					currentPersonToCheck.Relationships[currentPerson.ID] = *relationship
				}
			}
		}

		currentGeneration := currentPerson.Generation
		personsToCheckRelationship, ok = personByGeneration[currentGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				relationship := currentPersonToCheck.FindCurrentGenerationRelationships(*currentPerson)
				if relationship != nil {
					currentPersonToCheck.Relationships[currentPerson.ID] = *relationship
				}
			}
		}

		previousGeneration := currentPerson.Generation - 1
		personsToCheckRelationship, ok = personByGeneration[previousGeneration]
		if ok {
			for _, currentPersonToCheck := range personsToCheckRelationship {
				relationship := currentPersonToCheck.FindNextGenerationRelationships(*currentPerson)
				if relationship != nil {
					currentPersonToCheck.Relationships[currentPerson.ID] = *relationship
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

type PersonJumps struct {
	Person         *Person
	JumpsUntilHere uint
}

func (fg *FamilyGraph) BaconsNumber(personIDA string, personIDB string) *uint {
	if personIDA == personIDB {
		zero := uint(0)
		return &zero
	}

	personA, ok := fg.Members[personIDA]
	if !ok {
		return nil
	}

	personB, ok := fg.Members[personIDB]
	if !ok {
		return nil
	}

	currentPerson := &PersonJumps{Person: personA, JumpsUntilHere: 0}
	visitPersonQueue := list.New()
	personAlreadyOnQueue := make(map[string]bool)
	personAlreadyOnQueue[currentPerson.Person.ID] = true

	var minimumJumps *uint
	// BSF to find the shortest path
	for currentPerson != nil || visitPersonQueue.Len() > 0 {

		if currentPerson.Person.ID == personB.ID {
			minimumJumps = &currentPerson.JumpsUntilHere
			break
		}

		for _, currentSpouse := range currentPerson.Person.Spouses {
			if _, ok := personAlreadyOnQueue[currentSpouse.ID]; !ok {
				visitPersonQueue.PushBack(&PersonJumps{Person: currentSpouse, JumpsUntilHere: currentPerson.JumpsUntilHere + 1})
				personAlreadyOnQueue[currentSpouse.ID] = true
			}
		}

		for _, currentParent := range currentPerson.Person.Parents {
			if _, ok := personAlreadyOnQueue[currentParent.ID]; !ok {
				visitPersonQueue.PushBack(&PersonJumps{Person: currentParent, JumpsUntilHere: currentPerson.JumpsUntilHere + 1})
				personAlreadyOnQueue[currentParent.ID] = true
			}
		}

		for _, currentChildren := range currentPerson.Person.Children {
			if _, ok := personAlreadyOnQueue[currentChildren.ID]; !ok {
				visitPersonQueue.PushBack(&PersonJumps{Person: currentChildren, JumpsUntilHere: currentPerson.JumpsUntilHere + 1})
				personAlreadyOnQueue[currentChildren.ID] = true
			}
		}

		nextPerson := visitPersonQueue.Front()
		if nextPerson != nil {
			visitPersonQueue.Remove(nextPerson)
			person := nextPerson.Value.(*PersonJumps)
			currentPerson = person
		} else {
			currentPerson = nil
		}
	}

	return minimumJumps
}
