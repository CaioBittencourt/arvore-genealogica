package controller

import (
	"context"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/repository"
)

type PersonController interface {
	GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error)
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

func (s personController) GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error) {
	familyGraph, err := s.personRepository.GetPersonFamilyGraphByID(ctx, personID, nil)
	if err != nil {
		return nil, err
	}

	if err := familyGraph.PopulateWithFamilyRelationships(personID); err != nil {
		return nil, err
	}

	return familyGraph, err
}

func (s personController) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	insertedPerson, err := s.personRepository.Store(ctx, person)
	if err != nil {
		return nil, err
	}

	return insertedPerson, err
}
