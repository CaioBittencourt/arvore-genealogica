package controller

import (
	"context"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/repository"
)

type PersonController interface {
	GetFamilyTreeByPersonID(ctx context.Context, personID string) (*domain.FamilyTree, error)
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

func (s personController) GetFamilyTreeByPersonID(ctx context.Context, personID string) (*domain.FamilyTree, error) {
	familyTree, err := s.personRepository.GetPersonFamilyTreeByID(ctx, personID, nil)
	return familyTree, err
}

func (s personController) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	insertedPerson, err := s.personRepository.Store(ctx, person)
	if err != nil {
		return nil, err
	}

	return insertedPerson, err
}
