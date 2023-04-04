package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
	databaseName = "familyTree"

	relativesFieldName = "relatives"
	depthFieldName     = "depthField"

	personCollectionName = "person"
)

type PersonWithRelatives struct {
	Person    `bson:"inline"`
	Relatives []Person `bson:"relatives,omitempty"`
}

type Person struct {
	ID          primitive.ObjectID   `bson:"_id,omitempty"`
	Name        string               `bson:"name,omitempty"`
	Gender      string               `bson:"gender,omitempty"`
	ParentIDS   []primitive.ObjectID `bson:"parentIds,omitempty"`
	ChildrenIDS []primitive.ObjectID `bson:"childrenIds,omitempty"`
	SpouseIDS   []primitive.ObjectID `bson:"spouseIds,omitempty"`
	Spouses     []Person             `bson:"spouses,omitempty"`
	Parents     []Person             `bson:"parents,omitempty"`
	Children    []Person             `bson:"children,omitempty"`
	DepthField  uint                 `bson:"depthField,omitempty"`
}

type PersonRepository struct {
	client mongo.Client
}

func NewPersonRepository(client mongo.Client) PersonRepository {
	return PersonRepository{client: client}
}

func buildDomainPersonFromRepositoryPerson(personRepository Person) domain.Person {
	person := domain.Person{
		ID:     personRepository.ID.Hex(),
		Name:   personRepository.Name,
		Gender: domain.GenderType(personRepository.Gender),
	}

	if len(personRepository.Parents) > 0 {
		for _, parent := range personRepository.Parents {
			domainParent := buildDomainPersonFromRepositoryPerson(parent)
			person.Parents = append(person.Parents, &domainParent)
		}
	} else {
		for _, parentID := range personRepository.ParentIDS {
			person.Parents = append(person.Parents, &domain.Person{ID: parentID.Hex()})
		}
	}

	if len(personRepository.Children) > 0 {
		for _, currentChildren := range personRepository.Children {
			domainChildren := buildDomainPersonFromRepositoryPerson(currentChildren)
			person.Children = append(person.Children, &domainChildren)
		}
	} else {
		for _, childrenID := range personRepository.ChildrenIDS {
			person.Children = append(person.Children, &domain.Person{ID: childrenID.Hex()})
		}
	}

	return person
}

func buildDomainPersonsFromRepositoryPersons(personsRepository []Person) []domain.Person {
	var domainPersons []domain.Person
	for _, personRepository := range personsRepository {
		domainPersons = append(domainPersons, buildDomainPersonFromRepositoryPerson(personRepository))
	}

	return domainPersons
}

func buildFamilyGraphFromPersonRelatives(personWithRelatives PersonWithRelatives) domain.FamilyGraph {
	personRelativesMap := make(map[string]Person, len(personWithRelatives.Relatives)+1)

	searchedPersonID := personWithRelatives.ID.Hex()
	personRelativesMap[searchedPersonID] = Person{
		ID:          personWithRelatives.ID,
		Name:        personWithRelatives.Name,
		Gender:      personWithRelatives.Gender,
		ParentIDS:   personWithRelatives.ParentIDS,
		ChildrenIDS: personWithRelatives.ChildrenIDS,
	}
	for _, relatedPerson := range personWithRelatives.Relatives {
		personRelativesMap[relatedPerson.ID.Hex()] = relatedPerson
	}

	graphMembersMapped := make(map[string]*domain.Person)
	buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[searchedPersonID], personRelativesMap, graphMembersMapped, 0)

	return domain.FamilyGraph{Members: graphMembersMapped}
}

// TODO: Move this to domain. Build the graph with person with relatives from the db.
func buildFamilyRelationshipsFromPersonRelatives(
	personRepository Person,
	personRelativesMap map[string]Person,
	graphMembersMapped map[string]*domain.Person,
	generation int) *domain.Person {

	if _, ok := graphMembersMapped[personRepository.ID.Hex()]; ok {
		return graphMembersMapped[personRepository.ID.Hex()]
	}

	person := domain.Person{
		ID:         personRepository.ID.Hex(),
		Name:       personRepository.Name,
		Gender:     domain.GenderType(personRepository.Gender),
		Generation: generation,
	}

	graphMembersMapped[person.ID] = &person

	for _, spouseObjectID := range personRepository.SpouseIDS {
		spouseID := spouseObjectID.Hex()
		if _, ok := personRelativesMap[spouseID]; !ok {
			continue
		}

		spouse := buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[spouseID], personRelativesMap, graphMembersMapped, generation)
		person.Spouses = append(person.Spouses, spouse)
	}

	for _, parentObjectID := range personRepository.ParentIDS {
		parentID := parentObjectID.Hex()
		if _, ok := personRelativesMap[parentID]; !ok {
			continue
		}

		parent := buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[parentID], personRelativesMap, graphMembersMapped, generation+1)
		person.Parents = append(person.Parents, parent)
	}

	for _, childrenObjectID := range personRepository.ChildrenIDS {
		childrenID := childrenObjectID.Hex()
		if _, ok := personRelativesMap[childrenID]; !ok {
			continue
		}

		children := buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[childrenID], personRelativesMap, graphMembersMapped, generation-1)
		person.Children = append(person.Children, children)
	}

	return &person
}

func convertIDStringToObjectsIDS(personIDS []string) ([]primitive.ObjectID, error) {
	var objectIDS []primitive.ObjectID
	for _, personID := range personIDS {
		personObjectId, err := primitive.ObjectIDFromHex(personID)
		if err != nil {
			return nil, err
		}
		objectIDS = append(objectIDS, personObjectId)
	}

	return objectIDS, nil
}

func (pr PersonRepository) graphlookupGetPersonRelativesByPersonIDS(ctx context.Context, personIDS []string, connectFromField string, maxDepth *int) ([]PersonWithRelatives, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(personIDS)
	if err != nil {
		return nil, err
	}

	matchStage := bson.D{{"$match", bson.D{{"_id", bson.D{{"$in", objectIDS}}}}}}
	graphLookupParameters := bson.D{
		{"from", personCollectionName},
		{"startWith", fmt.Sprintf("$%s", connectFromField)},
		{"connectFromField", connectFromField},
		{"connectToField", "_id"},
		{"as", relativesFieldName},
		{"depthField", depthFieldName},
	}

	if maxDepth != nil {
		graphLookupParameters = append(graphLookupParameters, bson.E{"maxDepth", *maxDepth})
	}

	graphLookupStage := bson.D{{"$graphLookup", graphLookupParameters}}
	pipelineStages := bson.A{
		matchStage,
		graphLookupStage,
	}

	cursor, err := personCollection.Aggregate(ctx, pipelineStages)
	if err != nil {
		return nil, err
	}

	var personRelatives []PersonWithRelatives
	if err := cursor.All(ctx, &personRelatives); err != nil {
		if err := cursor.Err(); err != nil {
			log.WithError(err).Warn("cursor: failed to fetch family tree by person id")
			return nil, err
		}
	}

	return personRelatives, nil
}

func mergeRelativesIntoSet(relativesLists [][]Person) []Person {
	relativesSet := make(map[string]Person)
	for _, relativesList := range relativesLists {
		for _, relativePerson := range relativesList {
			_, ok := relativesSet[relativePerson.ID.Hex()]
			if !ok {
				relativesSet[relativePerson.ID.Hex()] = relativePerson
			}
		}
	}

	relatives := make([]Person, len(relativesSet))
	for _, person := range relativesSet {
		relatives = append(relatives, person)
	}

	return relatives
}

func (pr PersonRepository) getDescendantsFromPersonWithAscendants(ctx context.Context, personWithAscendants PersonWithRelatives, maxDepth uint) ([][]Person, error) {
	var relativesLists [][]Person

	//NOTE: find brothers, cousins and etc. If we wanted to find more relations we could go higher on the ascendant. but since the challenge goes only to that level...
	var ascendantsIDS []string
	for _, person := range personWithAscendants.Relatives {
		if person.DepthField <= maxDepth {
			ascendantsIDS = append(ascendantsIDS, person.ID.Hex())
		}
	}

	if len(ascendantsIDS) > 0 {
		depth := 1 //NOTE: get brothers, cousins, childrens, this could be configurable
		parentAndGrandParentsWithRelatives, err := pr.graphlookupGetPersonRelativesByPersonIDS(ctx, ascendantsIDS, "childrenIds", &depth)
		if err != nil {
			return nil, err
		}

		for _, parentAndGrandparent := range parentAndGrandParentsWithRelatives {
			relativesLists = append(relativesLists, parentAndGrandparent.Relatives)
		}
	} else {
		depth := 0 //NOTE: didnt find any ascendant, use my id to fetch childrens
		myRelatives, err := pr.graphlookupGetPersonRelativesByPersonIDS(ctx, []string{personWithAscendants.ID.Hex()}, "childrenIds", &depth)
		if err != nil {
			return nil, err
		}

		for _, myRelatives := range myRelatives {
			relativesLists = append(relativesLists, myRelatives.Relatives)
		}
	}

	return relativesLists, nil
}

func (pr PersonRepository) GetPersonFamilyGraphByID(ctx context.Context, personID string, maxDepth *int64) (*domain.FamilyGraph, error) {
	result, err := pr.graphlookupGetPersonRelativesByPersonIDS(ctx, []string{personID}, "parentIds", nil)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("unable to find person with that id")
	}

	personWithAscendants := result[0]

	var relativesLists [][]Person
	relativesLists = append(relativesLists, personWithAscendants.Relatives)

	//NOTE: max depth 1, means that is only going to search for parents and grandparents
	listOfDescendantsRelatives, err := pr.getDescendantsFromPersonWithAscendants(ctx, personWithAscendants, 1)
	if err != nil {
		return nil, err
	}

	for _, descendantsRelatives := range listOfDescendantsRelatives {
		relativesLists = append(relativesLists, descendantsRelatives)
	}

	personWithRelatives := personWithAscendants

	spouseIds := []string{personWithRelatives.ID.Hex()}
	for _, spouseId := range personWithRelatives.SpouseIDS {
		spouseIds = append(spouseIds, spouseId.Hex())
	}

	for _, relative := range personWithRelatives.Relatives {
		for _, spouseId := range relative.SpouseIDS {
			spouseIds = append(spouseIds, spouseId.Hex())
		}
	}

	spouses, err := pr.getPersonsByIDS(ctx, spouseIds)
	if err != nil {
		return nil, err
	}
	relativesLists = append(relativesLists, spouses)

	personWithRelatives.Relatives = mergeRelativesIntoSet(relativesLists)

	familyGraph := buildFamilyGraphFromPersonRelatives(personWithRelatives)

	return &familyGraph, nil
}

func (pr PersonRepository) addChildrenToParents(ctx context.Context, parentIDS []string, childrenID string) error {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	childrenObjectID, err := primitive.ObjectIDFromHex(childrenID)
	if err != nil {
		return err
	}

	parentsObjectIDS, err := convertIDStringToObjectsIDS(parentIDS)
	if err != nil {
		return err
	}

	if _, err := personCollection.UpdateMany(ctx,
		bson.D{{"_id", bson.D{{"$in", parentsObjectIDS}}}},
		bson.D{{"$addToSet", bson.D{{"childrenIds", childrenObjectID}}}},
	); err != nil {
		return err
	}

	return nil
}

func (pr PersonRepository) storePerson(ctx context.Context, person domain.Person) (*primitive.ObjectID, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	childrenIDs := make([]primitive.ObjectID, len(person.Children))
	for _, children := range person.Children {
		childrenObjectId, err := primitive.ObjectIDFromHex(children.ID)
		if err != nil {
			return nil, err
		}
		childrenIDs = append(childrenIDs, childrenObjectId)
	}

	parentIDS := []primitive.ObjectID{}
	for _, parent := range person.Parents {
		parentObjectID, err := primitive.ObjectIDFromHex(parent.ID)
		if err != nil {
			return nil, err
		}

		parentIDS = append(parentIDS, parentObjectID)
	}

	spouseIDS := []primitive.ObjectID{}
	for _, spouse := range person.Spouses {
		spouseObjectID, err := primitive.ObjectIDFromHex(spouse.ID)
		if err != nil {
			return nil, err
		}

		spouseIDS = append(spouseIDS, spouseObjectID)
	}

	insertedPerson, err := personCollection.InsertOne(ctx, bson.D{
		{"name", person.Name},
		{"gender", person.Gender},
		{"parentIds", parentIDS},
		{"childrenIds", childrenIDs},
		{"spouseIds", spouseIDS},
	})
	if err != nil {
		return nil, err
	}

	insertedPersonObjectID, ok := insertedPerson.InsertedID.(primitive.ObjectID)
	if !ok {
		return nil, errors.New("failed to convert inserted document to Object ID")
	}

	return &insertedPersonObjectID, nil
}
func (pr PersonRepository) getPersonWithImmediateRelativesByIDS(ctx context.Context, personIDS []string) ([]Person, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(personIDS)
	if err != nil {
		return nil, err
	}

	matchStage := bson.D{{"$match", bson.D{{"_id", bson.D{{"$in", objectIDS}}}}}}

	lookupParentsStage := bson.D{{"$lookup", bson.D{
		{"from", personCollectionName},
		{"localField", "parentIds"},
		{"foreignField", "_id"},
		{"as", "parents"},
	}}}

	lookupChildrenStage := bson.D{{"$lookup", bson.D{
		{"from", personCollectionName},
		{"localField", "childrenIds"},
		{"foreignField", "_id"},
		{"as", "children"},
	}}}

	pipelineStages := bson.A{
		matchStage,
		lookupParentsStage,
		lookupChildrenStage,
	}

	cursor, err := personCollection.Aggregate(ctx, pipelineStages)
	if err != nil {
		return nil, err
	}

	var persons []Person
	if err := cursor.All(ctx, &persons); err != nil {
		if err := cursor.Err(); err != nil {
			log.WithError(err).Warn("cursor: failed to fetch person with immediate relatives by person id")
			return nil, err
		}
	}

	return persons, nil
}

func (pr PersonRepository) getPersonsByIDS(ctx context.Context, IDS []string) ([]Person, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(IDS)
	if err != nil {
		return nil, err
	}

	cursor, err := personCollection.Find(ctx, bson.D{{"_id", bson.D{{"$in", objectIDS}}}})
	if err != nil {
		return nil, err
	}

	var persons []Person
	if err := cursor.All(ctx, &persons); err != nil {
		if err := cursor.Err(); err != nil {
			log.WithError(err).Warn("cursor: failed to get persons by ids")
			return nil, err
		}
	}

	return persons, nil
}

func (pr PersonRepository) getPersonsByChildrenIDS(ctx context.Context, IDS []string) ([]Person, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(IDS)
	if err != nil {
		return nil, err
	}

	cursor, err := personCollection.Find(ctx, bson.D{{"childrenIds", bson.D{{"$in", objectIDS}}}})
	if err != nil {
		return nil, err
	}

	var persons []Person
	if err := cursor.All(ctx, &persons); err != nil {
		if err := cursor.Err(); err != nil {
			log.WithError(err).Warn("cursor: failed to get persons by ids")
			return nil, err
		}
	}

	return persons, nil
}

func (pr PersonRepository) addSpouseToPersons(ctx context.Context, personsIDS []string, spouseID string) (*mongo.UpdateResult, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	spouseObjectID, err := primitive.ObjectIDFromHex(spouseID)
	if err != nil {
		return nil, err
	}

	personObjectIDS, err := convertIDStringToObjectsIDS(personsIDS)
	if err != nil {
		return nil, err
	}

	result, err := personCollection.UpdateMany(ctx,
		bson.D{{"_id", bson.D{{"$in", personObjectIDS}}}},
		bson.D{{"$addToSet", bson.D{{"spouseIds", spouseObjectID}}}},
	)

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (pr PersonRepository) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		var childrenIDS []string
		for _, children := range person.Children {
			childrenIDS = append(childrenIDS, children.ID)
		}

		var spouses []Person
		var err error
		if len(childrenIDS) > 0 {
			spouses, err = pr.getPersonsByChildrenIDS(ctx, childrenIDS)
			if err != nil {
				return nil, err
			}
		}

		var spousesIds []string
		if len(spouses) > 0 {
			for _, currentSpouse := range spouses {
				spouse := buildDomainPersonFromRepositoryPerson(currentSpouse)
				person.Spouses = append(person.Spouses, &spouse)
				spousesIds = append(spousesIds, spouse.ID)
			}
		}

		insertedPersonObjectID, err := pr.storePerson(sessCtx, person)
		if err != nil {
			return nil, err
		}

		if len(spousesIds) > 0 {
			if _, err := pr.addSpouseToPersons(ctx, spousesIds, insertedPersonObjectID.Hex()); err != nil {
				return nil, err
			}
		}

		var parentsIDS []string
		for _, parent := range person.Parents {
			parentsIDS = append(parentsIDS, parent.ID)
		}

		if len(parentsIDS) == 2 {
			result, err := pr.addSpouseToPersons(ctx, []string{parentsIDS[1]}, parentsIDS[0])
			if err != nil {
				return nil, err
			}

			if result.ModifiedCount > 0 {
				if _, err := pr.addSpouseToPersons(ctx, []string{parentsIDS[0]}, parentsIDS[1]); err != nil {
					return nil, err
				}
			}
		}

		if len(parentsIDS) > 0 {
			if err := pr.addChildrenToParents(sessCtx, parentsIDS, insertedPersonObjectID.Hex()); err != nil {
				return nil, err
			}
		}

		return insertedPersonObjectID, nil
	}

	session, err := pr.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)
	insertedID, err := session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, err
	}

	insertedPersonObjectID, ok := insertedID.(*primitive.ObjectID)
	if !ok {
		return nil, errors.New("failed to convert inserted document to Object ID")
	}

	persons, err := pr.getPersonWithImmediateRelativesByIDS(ctx, []string{insertedPersonObjectID.Hex()})
	if err != nil {
		return nil, err
	}

	if len(persons) == 0 {
		return nil, errors.New("unable to find inserted person")
	}

	domainPersons := buildDomainPersonsFromRepositoryPersons(persons)
	return &domainPersons[0], nil
}
