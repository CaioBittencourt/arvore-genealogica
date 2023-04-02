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
	ID         primitive.ObjectID   `bson:"_id,omitempty"`
	Name       string               `bson:"name,omitempty"`
	Gender     string               `bson:"gender,omitempty"`
	Parents    []primitive.ObjectID `bson:"parents,omitempty"`
	Children   []primitive.ObjectID `bson:"children,omitempty"`
	DepthField uint                 `bson:"depthField,omitempty"`
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
		person.Parents = append(person.Parents, &domain.Person{ID: parent.Hex()})
	}

	for _, children := range personRepository.Children {
		person.Children = append(person.Children, &domain.Person{ID: children.Hex()})
	}

	return person
}

func buildFamilyGraphFromPersonRelatives(personWithRelatives PersonWithRelatives) domain.FamilyGraph {
	personRelativesMap := make(map[string]Person, len(personWithRelatives.Relatives)+1)

	searchedPersonID := personWithRelatives.ID.Hex()
	personRelativesMap[searchedPersonID] = Person{
		ID:       personWithRelatives.ID,
		Name:     personWithRelatives.Name,
		Gender:   personWithRelatives.Gender,
		Parents:  personWithRelatives.Parents,
		Children: personWithRelatives.Children,
	}
	for _, relatedPerson := range personWithRelatives.Relatives {
		personRelativesMap[relatedPerson.ID.Hex()] = relatedPerson
	}

	graphMembersMapped := make(map[string]*domain.Person)
	buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[searchedPersonID], personRelativesMap, graphMembersMapped, 0)

	return domain.FamilyGraph{Members: graphMembersMapped}
}

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

	for _, parentObjectID := range personRepository.Parents {
		parentID := parentObjectID.Hex()
		if _, ok := personRelativesMap[parentID]; !ok {
			continue
		}

		parent := buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[parentID], personRelativesMap, graphMembersMapped, generation+1)
		person.Parents = append(person.Parents, parent)
	}

	for _, childrenObjectID := range personRepository.Children {
		childrenID := childrenObjectID.Hex()
		if _, ok := personRelativesMap[childrenID]; !ok {
			continue
		}

		children := buildFamilyRelationshipsFromPersonRelatives(personRelativesMap[childrenID], personRelativesMap, graphMembersMapped, generation-1)
		person.Children = append(person.Children, children)
	}

	return &person
}

func buildFamilyGraphFromPersonWithRelatives(ctx context.Context, personWithRelatives PersonWithRelatives) (domain.Person, error) {
	personRelativesMap := make(map[string]*domain.Person, len(personWithRelatives.Relatives)+1)
	person := domain.Person{
		ID:     personWithRelatives.ID.Hex(),
		Name:   personWithRelatives.Name,
		Gender: domain.GenderType(personWithRelatives.Gender),
	}

	for _, parent := range personWithRelatives.Parents {
		person.Parents = append(person.Parents, &domain.Person{ID: parent.Hex()})
	}

	for _, children := range personWithRelatives.Children {
		person.Children = append(person.Children, &domain.Person{ID: children.Hex()})
	}

	personRelativesMap[person.ID] = &person
	for _, relatedPerson := range personWithRelatives.Relatives {
		person := buildDomainPersonFromRepositoryPerson(relatedPerson)
		personRelativesMap[person.ID] = &person
	}

	for _, currentPerson := range personRelativesMap {
		for i, emptyParent := range currentPerson.Parents {
			parent := personRelativesMap[emptyParent.ID]
			if parent != nil {
				currentPerson.Parents[i] = parent
			}
		}

		for i, emptyChildren := range currentPerson.Children {
			children := personRelativesMap[emptyChildren.ID]
			if children != nil {
				currentPerson.Children[i] = children
			}
		}
	}

	return person, nil

}

func (pr PersonRepository) graphlookupGetPersonRelatives(ctx context.Context, personIDS []string, connectFromField string, maxDepth *int) ([]PersonWithRelatives, error) {
	personCollection := pr.client.Database(databaseName).Collection(personCollectionName)

	var objectIDS []primitive.ObjectID
	for _, personID := range personIDS {
		personObjectId, err := primitive.ObjectIDFromHex(personID)
		if err != nil {
			return nil, err
		}
		objectIDS = append(objectIDS, personObjectId)
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

func (pr PersonRepository) GetPersonFamilyTreeByID(ctx context.Context, personID string, maxDepth *int64) (*domain.FamilyGraph, error) {
	result, err := pr.graphlookupGetPersonRelatives(ctx, []string{personID}, "parents", nil)
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("unable to find person with that id")
	}

	personWithAscendants := result[0]

	relativesLists := make([][]Person, len(personWithAscendants.Parents))
	relativesLists = append(relativesLists, personWithAscendants.Relatives)

	//NOTE: find brothers, cousins and etc. If we wanted to find more relations we could go higher on the ascendant. but since the challenge goes only to that level...
	var parentsAndGrandParentsIDS []string
	for _, person := range personWithAscendants.Relatives {
		if person.DepthField == 0 || person.DepthField == 1 {
			parentsAndGrandParentsIDS = append(parentsAndGrandParentsIDS, person.ID.Hex())
		}
	}

	depth := 1 // get brothers and cousins, this could be configurable
	parentAndGrandParentsWithRelatives, err := pr.graphlookupGetPersonRelatives(ctx, parentsAndGrandParentsIDS, "children", &depth)
	if err != nil {
		return nil, err
	}

	for _, parentAndGrandparent := range parentAndGrandParentsWithRelatives {
		relativesLists = append(relativesLists, parentAndGrandparent.Relatives)
	}

	personWithRelatives := personWithAscendants
	personWithRelatives.Relatives = mergeRelativesIntoSet(relativesLists)

	familyTree := buildFamilyGraphFromPersonRelatives(personWithRelatives)

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
