package mongodb

import (
	"context"
	"errors"
	"fmt"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const (
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
	Parents     []Person             `bson:"parents,omitempty"`
	Children    []Person             `bson:"children,omitempty"`
	DepthField  uint                 `bson:"depthField,omitempty"`
}

type PersonRepository struct {
	client       mongo.Client
	databaseName string
}

func NewPersonRepository(client mongo.Client, databaseName string) PersonRepository {
	return PersonRepository{client: client, databaseName: databaseName}
}

func buildRepositoryPersonFromDomainPerson(domainPerson domain.Person) Person {
	repositoryPerson := Person{
		Name:        domainPerson.Name,
		Gender:      string(domainPerson.Gender),
		ChildrenIDS: []primitive.ObjectID{},
		ParentIDS:   []primitive.ObjectID{},
	}

	if domainPerson.ID != "" {
		objectID, err := primitive.ObjectIDFromHex(domainPerson.ID)
		if err == nil {
			repositoryPerson.ID = objectID
		}
	}

	for _, parent := range domainPerson.Parents {
		objectID, err := primitive.ObjectIDFromHex(parent.ID)
		if err == nil {
			repositoryPerson.ParentIDS = append(repositoryPerson.ParentIDS, objectID)
		}
	}

	for _, children := range domainPerson.Children {
		objectID, err := primitive.ObjectIDFromHex(children.ID)
		if err == nil {
			repositoryPerson.ChildrenIDS = append(repositoryPerson.ChildrenIDS, objectID)
		}
	}

	return repositoryPerson
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
		mappedChildren, ok := personRelativesMap[childrenID]
		if !ok {
			continue
		}

		if len(mappedChildren.ParentIDS) == 2 {
			spouse1, ok1 := graphMembersMapped[mappedChildren.ParentIDS[0].Hex()]
			spouse2, ok2 := graphMembersMapped[mappedChildren.ParentIDS[1].Hex()]
			// add spouses
			if ok1 && ok2 {
				var currentPersonSpouse *domain.Person
				var spouseForCurrentPerson *domain.Person
				if person.ID == spouse1.ID {
					currentPersonSpouse = spouse1
					spouseForCurrentPerson = spouse2
				} else {
					currentPersonSpouse = spouse2
					spouseForCurrentPerson = spouse1
				}

				if !currentPersonSpouse.HasSpouse(*spouseForCurrentPerson) {
					currentPersonSpouse.Spouses = append(currentPersonSpouse.Spouses, spouseForCurrentPerson)
				}
			}
		}

		children := buildFamilyRelationshipsFromPersonRelatives(mappedChildren, personRelativesMap, graphMembersMapped, generation-1)
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
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(personIDS)
	if err != nil {
		return nil, err
	}

	matchStage := bson.M{"$match": bson.M{"_id": bson.M{"$in": objectIDS}}}
	graphLookupParameters := bson.M{
		"from":             personCollectionName,
		"startWith":        fmt.Sprintf("$%s", connectFromField),
		"connectFromField": connectFromField,
		"connectToField":   "_id",
		"as":               relativesFieldName,
		"depthField":       depthFieldName,
	}

	if maxDepth != nil {
		graphLookupParameters["maxDepth"] = *maxDepth
	}

	graphLookupStage := bson.M{"$graphLookup": graphLookupParameters}
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
		return nil, nil
	}

	personWithAscendants := result[0]

	var relativesLists [][]Person
	relativesLists = append(relativesLists, personWithAscendants.Relatives)

	//NOTE: max depth 1, means that is only going to search for parents and grandparents
	listOfDescendantsRelatives, err := pr.getDescendantsFromPersonWithAscendants(ctx, personWithAscendants, 1)
	if err != nil {
		return nil, err
	}

	relativesLists = append(relativesLists, listOfDescendantsRelatives...)

	personWithRelatives := personWithAscendants

	if len(personWithRelatives.ChildrenIDS) > 0 {
		personsByChildrenID, err := pr.getPersonsByChildrenIDS(ctx, personWithRelatives.ChildrenIDS)
		if err != nil {
			return nil, err
		}

		var spouses []Person
		for _, person := range personsByChildrenID {
			if person.ID.Hex() != personID {
				spouses = append(spouses, person)
			}
		}
		relativesLists = append(relativesLists, spouses)
	}

	personWithRelatives.Relatives = mergeRelativesIntoSet(relativesLists)

	familyGraph := buildFamilyGraphFromPersonRelatives(personWithRelatives)

	return &familyGraph, nil
}

func (pr PersonRepository) updateParentsForInsertedPerson(ctx context.Context, parentsObjectIDS []primitive.ObjectID, childrenObjectID primitive.ObjectID) error {
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	if _, err := personCollection.UpdateMany(ctx,
		bson.M{"_id": bson.M{"$in": parentsObjectIDS}},
		bson.M{"$addToSet": bson.M{"childrenIds": childrenObjectID}},
	); err != nil {
		return err
	}

	return nil
}

func (pr PersonRepository) storePerson(ctx context.Context, person Person) (*primitive.ObjectID, error) {
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	insertedPerson, err := personCollection.InsertOne(ctx, bson.M{
		"name":        person.Name,
		"gender":      person.Gender,
		"parentIds":   person.ParentIDS,
		"childrenIds": person.ChildrenIDS,
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
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	objectIDS, err := convertIDStringToObjectsIDS(personIDS)
	if err != nil {
		return nil, err
	}

	matchStage := bson.M{"$match": bson.M{"_id": bson.M{"$in": objectIDS}}}

	lookupParentsStage := bson.M{"$lookup": bson.M{
		"from":         personCollectionName,
		"localField":   "parentIds",
		"foreignField": "_id",
		"as":           "parents",
	}}

	lookupChildrenStage := bson.M{"$lookup": bson.M{
		"from":         personCollectionName,
		"localField":   "childrenIds",
		"foreignField": "_id",
		"as":           "children",
	}}

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
			return nil, err
		}
	}

	return persons, nil
}

func (pr PersonRepository) GetPersonWithImmediateRelativesByIDS(ctx context.Context, personIDS []string) ([]domain.Person, error) {
	persons, err := pr.getPersonWithImmediateRelativesByIDS(ctx, personIDS)
	if err != nil {
		return nil, err
	}

	return buildDomainPersonsFromRepositoryPersons(persons), nil
}

func (pr PersonRepository) getPersonsByChildrenIDS(ctx context.Context, objectIDS []primitive.ObjectID) ([]Person, error) {
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	cursor, err := personCollection.Find(ctx, bson.M{"childrenIds": bson.M{"$in": objectIDS}})
	if err != nil {
		return nil, err
	}

	var persons []Person
	if err := cursor.All(ctx, &persons); err != nil {
		if err := cursor.Err(); err != nil {
			return nil, err
		}
	}

	return persons, nil
}

func (pr PersonRepository) updateChildrenForInsertedPerson(ctx context.Context, childrenObjectIDS []primitive.ObjectID, parentObjectID primitive.ObjectID) error {
	personCollection := pr.client.Database(pr.databaseName).Collection(personCollectionName)

	if len(childrenObjectIDS) > 1 {
		_, err := personCollection.UpdateMany(ctx,
			bson.M{"_id": bson.M{"$in": childrenObjectIDS}},
			bson.M{"$addToSet": bson.M{"parentIds": parentObjectID}},
		)

		if err != nil {
			return err
		}
	} else {
		if _, err := personCollection.UpdateOne(ctx,
			bson.M{"_id": childrenObjectIDS[0]},
			bson.M{"$addToSet": bson.M{"parentIds": parentObjectID}},
		); err != nil {
			return err
		}
	}

	return nil
}

func (pr PersonRepository) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	repositoryPerson := buildRepositoryPersonFromDomainPerson(person)

	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		insertedPersonObjectID, err := pr.storePerson(sessCtx, repositoryPerson)
		if err != nil {
			return nil, err
		}

		if len(repositoryPerson.ParentIDS) > 0 {
			if err := pr.updateParentsForInsertedPerson(sessCtx, repositoryPerson.ParentIDS, *insertedPersonObjectID); err != nil {
				return nil, err
			}
		}

		if len(repositoryPerson.ChildrenIDS) > 0 {
			if err := pr.updateChildrenForInsertedPerson(sessCtx, repositoryPerson.ChildrenIDS, *insertedPersonObjectID); err != nil {
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
