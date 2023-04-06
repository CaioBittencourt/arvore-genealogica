package controller

import (
	"context"
	"fmt"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/errors"
	"github.com/CaioBittencourt/arvore-genealogica/repository"
	log "github.com/sirupsen/logrus"
)

type PersonController interface {
	GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error)
	BaconsNumber(ctx context.Context, firstPersonID string, secondPersonID string) (*uint, error)
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
		log.WithError(err).Error("person: failed to get family graph by ID")
		return nil, err
	}

	if familyGraph == nil {
		return nil, errors.NewApplicationError("person not found", errors.PersonNotFoundErrorCode)
	}

	if err := familyGraph.PopulateFamilyWithRelationships(personID); err != nil {
		log.WithError(err).Error("person: failed to populate family with relationships")
		return nil, err
	}

	return familyGraph, err
}

func (pc personController) getBaconNumber(ctx context.Context, firstPersonID string, secondPersonID string) (*uint, error) {
	familyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, firstPersonID, nil)
	if err != nil {
		log.WithError(err).Error("person: failed to get family graph by ID")
		return nil, err
	}

	if familyGraph == nil {
		return nil, errors.NewApplicationError(fmt.Sprintf("person with id %s not found", firstPersonID), errors.PersonNotFoundErrorCode)
	}

	baconsNumber := familyGraph.BaconsNumber(firstPersonID, secondPersonID)
	if baconsNumber == nil {
		return nil, errors.NewApplicationError("persons dont belong to eachothers graph", errors.PersonNotFoundInGraph)
	}

	return baconsNumber, nil
}

func (pc personController) BaconsNumber(ctx context.Context, firstPersonID string, secondPersonID string) (*uint, error) {
	baconNumber, err := pc.getBaconNumber(ctx, firstPersonID, secondPersonID)
	if err != nil && !errors.ErrorHasCode(err, errors.PersonNotFoundInGraph) {
		return nil, err
	}

	if baconNumber != nil {
		return baconNumber, err
	}

	baconNumber, err = pc.getBaconNumber(ctx, secondPersonID, firstPersonID)
	return baconNumber, err

}

func (pc personController) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	if err := person.Validate(); err != nil {
		log.WithError(err).Error("person: validate failed for person to store")
		return nil, err
	}

	// incest: parent being inserted: my father or mother are in eachothers graphs?
	insertedPerson, err := pc.personRepository.Store(ctx, person)
	if err != nil {
		log.WithError(err).Error("person: failed to store person")
		return nil, err
	}

	return insertedPerson, err
}
