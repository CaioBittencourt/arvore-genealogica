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
	ID       primitive.ObjectID   `bson:"_id,omitempty"`
	Name     string               `bson:"name,omitempty"`
	Gender   string               `bson:"gender,omitempty"`
	Parents  []primitive.ObjectID `bson:"parents,omitempty"`
	Children []primitive.ObjectID `bson:"children,omitempty"`
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

	for _, parent := range personRepository.Parents {
		person.Parents = append(person.Parents, domain.Person{ID: parent.Hex()})
	}

	for _, children := range personRepository.Children {
		person.Children = append(person.Children, domain.Person{ID: children.Hex()})
	}

	return person
}

func buildFamilyMembersFromPersonRelatives(
	personRepository Person,
	visitedPersonsMap map[string]bool,
	personRelativesMap map[string]Person,
	generation int) domain.FamilyTreeMember {
	familyMember := domain.FamilyTreeMember{
		ID:         personRepository.ID.Hex(),
		Name:       personRepository.Name,
		Gender:     domain.GenderType(personRepository.Gender),
		Generation: generation,
	}

	for _, parentObjectID := range personRepository.Parents {
		parentID := parentObjectID.Hex()
		familyMember.ParentIDS = append(familyMember.ParentIDS, parentID)
		if _, ok := personRelativesMap[parentID]; !ok {
			continue
		}

		if !visitedPersonsMap[parentID] {
			visitedPersonsMap[parentID] = true
			familyMember.ParentToVisit = append(familyMember.ParentToVisit, buildFamilyMembersFromPersonRelatives(personRelativesMap[parentID], visitedPersonsMap, personRelativesMap, generation+1))
		}
	}

	for _, childrenObjectID := range personRepository.Children {
		childrenID := childrenObjectID.Hex()
		familyMember.ChildrenIDS = append(familyMember.ChildrenIDS, childrenID)
		if _, ok := personRelativesMap[childrenID]; !ok {
			continue
		}

		if !visitedPersonsMap[childrenID] {
			visitedPersonsMap[childrenID] = true
			familyMember.ParentToVisit = append(familyMember.ParentToVisit, buildFamilyMembersFromPersonRelatives(personRelativesMap[childrenID], visitedPersonsMap, personRelativesMap, generation-1))
		}
	}

	return familyMember
}

func buildPersonFromPersonWithRelatives(ctx context.Context, personWithRelatives PersonWithRelatives) domain.FamilyTree {
	personRelativesMap := make(map[string]Person, len(personWithRelatives.Relatives)+1)

	rootPersonID := personWithRelatives.ID.Hex()
	personRelativesMap[rootPersonID] = Person{
		ID:       personWithRelatives.ID,
		Name:     personWithRelatives.Name,
		Gender:   personWithRelatives.Gender,
		Parents:  personWithRelatives.Parents,
		Children: personWithRelatives.Children,
	}
	for _, relatedPerson := range personWithRelatives.Relatives {
		// person := buildDomainPersonFromRepositoryPerson(relatedPerson)
		personRelativesMap[relatedPerson.ID.Hex()] = relatedPerson
	}

	fmt.Println(personRelativesMap)
	visitedPersonsMap := map[string]bool{rootPersonID: true}
	familyTree := domain.FamilyTree{
		Root: buildFamilyMembersFromPersonRelatives(personRelativesMap[rootPersonID], visitedPersonsMap, personRelativesMap, 0),
	}

	return familyTree
}

func (pr PersonRepository) GetPersonFamilyTreeByID(ctx context.Context, personID string, maxDepth *int64) (*domain.FamilyTree, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	personObjectId, err := primitive.ObjectIDFromHex(personID)
	if err != nil {
		return nil, err
	}

	matchStage := bson.D{{"$match", bson.D{{"_id", personObjectId}}}}
	graphLookupParameters := bson.D{
		{"from", personCollectionName},
		{"startWith", "$parents"},
		{"connectFromField", "parents"},
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

	if ok := cursor.Next(ctx); !ok {
		if err := cursor.Err(); err != nil {
			log.WithError(err).Warn("cursor: failed to fetch family tree by person id")
			return nil, err
		}
	}

	var personRelatives PersonWithRelatives
	err = cursor.Decode(&personRelatives)
	if err != nil {
		return nil, err
	}

	familyTree := buildPersonFromPersonWithRelatives(ctx, personRelatives)
	return &familyTree, nil
}

func (pr PersonRepository) GetDescendantsByPersonID(ctx context.Context, personID string, maxDepth *int64) ([]domain.Person, error) {
	return nil, nil
}

func (pr PersonRepository) addChildrenToParents(ctx context.Context, parentIDS []primitive.ObjectID, childrenID primitive.ObjectID) error {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	if _, err := personCollection.UpdateMany(ctx,
		bson.D{{"_id", bson.D{{"$in", parentIDS}}}},
		bson.D{{"$addToSet", bson.D{{"children", childrenID}}}},
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

	parents := []primitive.ObjectID{}
	for _, parent := range person.Parents {
		parentObjectID, err := primitive.ObjectIDFromHex(parent.ID)
		if err != nil {
			return nil, err
		}

		parents = append(parents, parentObjectID)
	}

	insertedPerson, err := personCollection.InsertOne(ctx, bson.D{
		{"name", person.Name},
		{"gender", person.Gender},
		{"parents", parents},
		{"children", childrenIDs},
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

// TODO: ADD CONSTRAINT TO WHEN THERE IS ALREADY A MOM OR DAD REGISTERED ON MONGO
func (pr PersonRepository) Store(ctx context.Context, person domain.Person) (*domain.Person, error) {
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		insertedPersonObjectID, err := pr.storePerson(sessCtx, person)
		if err != nil {
			return nil, err
		}

		parentsIDS := []primitive.ObjectID{}
		for _, parent := range person.Parents {
			parentObjectID, err := primitive.ObjectIDFromHex(parent.ID)
			if err != nil {
				return nil, err
			}

			parentsIDS = append(parentsIDS, parentObjectID)
		}

		pr.addChildrenToParents(sessCtx, parentsIDS, *insertedPersonObjectID)

		return insertedPersonObjectID, nil
	}

	session, err := pr.client.StartSession()
	if err != nil {
		return nil, err
	}
	defer session.EndSession(ctx)
	insertedPersonObjectID, err := session.WithTransaction(ctx, callback)
	if err != nil {
		return nil, err
	}

	insertedPerson := Person{}
	err = pr.client.
		Database(databaseName).
		Collection(personCollectionName).
		FindOne(ctx, bson.D{{"_id", insertedPersonObjectID}}).Decode(&insertedPerson)
	if err != nil {
		return nil, err
	}
	// decidir retorno da api de STORE

	return nil, nil
}
