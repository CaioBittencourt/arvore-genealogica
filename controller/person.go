package controller

import (
	"context"
	"errors"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/repository"
)

type PersonController interface {
	GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error)
	BaconsNumber(ctx context.Context, firstPersonID string, secondPersonID string) ([]domain.Person, *uint, error)
	Store(ctx context.Context, person domain.Person) (*domain.Person, error)
}

type personController struct {
	personRepository repository.PersonRepository
}

func NewPersonController(
	personRepository repository.PersonRepository,
) PersonController {
	return personController{
		personRepository: personRepository,
	}
}

func (pc personController) GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error) {
	familyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, personID, nil)
	if err != nil {
		return nil, err
	}

	if err := familyGraph.PopulateWithFamilyRelationships(personID); err != nil {
		return nil, err
	}

	return familyGraph, err
}

func (pc personController) BaconsNumber(ctx context.Context, firstPersonID string, secondPersonID string) ([]domain.Person, *uint, error) {
	var baconsNumber *uint

	familyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, firstPersonID, nil)
	if err != nil {
		return nil, nil, err
	}

	baconsNumber = familyGraph.BaconsNumber(firstPersonID, secondPersonID)
	if baconsNumber != nil {
		return []domain.Person{*familyGraph.Members[firstPersonID], *familyGraph.Members[secondPersonID]}, baconsNumber, nil
	}

	// NOTE: the graphs are different, i have to search in both graphs in case the first one doesnt have both members!
	secondFamilyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, secondPersonID, nil)
	if err != nil {
		return nil, nil, err
	}

	baconsNumber = secondFamilyGraph.BaconsNumber(firstPersonID, secondPersonID)
	if baconsNumber == nil {
		return nil, nil, nil
	}

	return []domain.Person{*secondFamilyGraph.Members[firstPersonID], *secondFamilyGraph.Members[secondPersonID]}, baconsNumber, nil
}

func (pc personController) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	if err := person.Validate(); err != nil {
		return nil, errors.New("invalid person to store")
	}

	// incest: parent being inserted: my father or mother are in eachothers graphs?
	insertedPerson, err := pc.personRepository.Store(ctx, person)
	if err != nil {
		return nil, err
	}

	return insertedPerson, err
}
