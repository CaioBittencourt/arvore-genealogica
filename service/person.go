package service

import (
	"context"
	"fmt"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"github.com/CaioBittencourt/arvore-genealogica/errors"
	"github.com/CaioBittencourt/arvore-genealogica/repository"
	log "github.com/sirupsen/logrus"
)

type PersonService interface {
	GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error)
	BaconsNumber(ctx context.Context, personAID string, personBID string) (*uint, error)
	GetRelationshipBetweenPersons(ctx context.Context, personAID string, personBID string) (*domain.Person, error)
	Store(ctx context.Context, person domain.Person) (*domain.Person, error)
}

type personService struct {
	personRepository repository.PersonRepository
}

func NewPersonService(
	personRepository repository.PersonRepository,
) PersonService {
	return personService{
		personRepository: personRepository,
	}
}

func (pc personService) GetFamilyGraphByPersonID(ctx context.Context, personID string) (*domain.FamilyGraph, error) {
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

func (pc personService) getBaconNumber(ctx context.Context, personAID string, personBID string) (*uint, error) {
	familyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, personAID, nil)
	if err != nil {
		log.WithError(err).Error("person: failed to get family graph by ID")
		return nil, err
	}

	if familyGraph == nil {
		return nil, errors.NewApplicationError(fmt.Sprintf("person with id %s not found", personAID), errors.PersonNotFoundErrorCode)
	}

	baconsNumber := familyGraph.BaconsNumber(personAID, personBID)
	if baconsNumber == nil {
		return nil, errors.NewApplicationError("persons dont belong to eachothers graph", errors.PersonNotFoundInGraph)
	}

	return baconsNumber, nil
}

func (pc personService) BaconsNumber(ctx context.Context, personAID string, personBID string) (*uint, error) {
	baconNumber, err := pc.getBaconNumber(ctx, personAID, personBID)
	if err != nil && !errors.ErrorHasCode(err, errors.PersonNotFoundInGraph) {
		return nil, err
	}

	if baconNumber != nil {
		return baconNumber, nil
	}

	baconNumber, err = pc.getBaconNumber(ctx, personBID, personAID)
	return baconNumber, err

}

func (pc personService) getRelationshipBetweenPersons(ctx context.Context, personAID string, personBID string) (*domain.Person, error) {
	familyGraph, err := pc.personRepository.GetPersonFamilyGraphByID(ctx, personAID, nil)
	if err != nil {
		log.WithError(err).Error("person: failed to get family graph by ID")
		return nil, err
	}

	if familyGraph == nil {
		return nil, errors.NewApplicationError(fmt.Sprintf("person with id %s not found", personAID), errors.PersonNotFoundErrorCode)
	}

	personWithRelationship := familyGraph.FindRelationshipBetweenPersons(personAID, personBID)
	if personWithRelationship == nil {
		return nil, errors.NewApplicationError("persons dont belong to eachothers graph", errors.PersonNotFoundInGraph)
	}

	return personWithRelationship, nil
}

func (pc personService) GetRelationshipBetweenPersons(ctx context.Context, personAID string, personBID string) (*domain.Person, error) {
	personWithRelationship, err := pc.getRelationshipBetweenPersons(ctx, personAID, personBID)
	if err != nil && !errors.ErrorHasCode(err, errors.PersonNotFoundInGraph) {
		return nil, err
	}

	if personWithRelationship != nil {
		return personWithRelationship, nil
	}

	personWithRelationship, err = pc.getRelationshipBetweenPersons(ctx, personBID, personAID)
	return personWithRelationship, err

}

func (pc personService) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	var childrens []domain.Person
	var err error
	if len(person.Children) > 0 {
		var childrenIDS []string
		for _, children := range person.Children {
			childrenIDS = append(childrenIDS, children.ID)
		}

		childrens, err = pc.personRepository.GetPersonWithImmediateRelativesByIDS(ctx, childrenIDS)
		if err != nil {
			log.WithError(err).Error("person: failed to store person")
			return nil, err
		}
	}

	if err := person.Validate(childrens); err != nil {
		log.WithError(err).Error("person: validate failed for person to store")
		return nil, err
	}

	for _, children := range childrens {
		mySpouseID := children.Parents[0].ID
		if len(children.Parents) == 1 {
			person.Spouses = append(person.Spouses, &domain.Person{ID: mySpouseID})
		}
	}

	// add relationship spouse between my parents if they dont have it already between them.
	spousesToInsert := map[string]string{}
	if len(person.Parents) == 2 {
		for i, parent := range person.Parents {
			var mySpouseID string
			if i == 0 {
				mySpouseID = person.Parents[i+1].ID
			} else {
				mySpouseID = person.Parents[i-1].ID
			}

			spousesToInsert[parent.ID] = mySpouseID
		}
	}

	insertedPerson, err := pc.personRepository.Store(ctx, person, spousesToInsert)
	if err != nil {
		log.WithError(err).Error("person: failed to store person")
		return nil, err
	}

	return insertedPerson, err
}
