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

// Fazendo duas buscas por pai e depois por filho eu não conseguiria achar Spouse ainda sim.

// Pegar irmãos com query mesmo pai
// Pegar sobrinhos com Query filho da minha irmã
// filhos
// Pegar Spouse com query
// O double mapping tem que rolar. COm a query de child eu consigo andar no grapho para baixo. E isso já resolve muita coisa, é só escolher o lugar certo para começar a descer.

// usar NewObjectID para criar um novo ID?

// Começar pelo meu pai / mãe e ir descendo (graphlookup) neles para identificar sobrinhos e irmãos. O dept, vai me dizer o que cada um é.

// Para pegar o Spouse podemos pegar todas as suas crianças e sobrinhos e fazer um graphlookup com eles juntos
//
// transversando um grapho eu consigo achar:
// Pais, Irmãos,
// {
//     "_id": ObjectId("642506fd04ac527b404e6bd1")
// }
// {
//     from: "person",
// //    startWith: {$eq: ["_id", ObjectId("642506fd04ac527b404e6bd1")]}, // connectToField value(s) that recursive search starts with
// //    startWith: {"id": "642506fd04ac527b404e6bd1"},
//     startWith: "$parents",
//     connectFromField: "parents",
//     connectToField: "id",
//     as: "hierarquia",
// //    maxDepth: 10, // optional
//     depthField: "depthField" // optional - name of field in output documents
//     //restrictSearchWithMatch: <document> // optional - specifying additional conditions for the recursive search
// }
//	{
//		"_id" : 1,
//		"name" : "Dev",
//		"reportingHierarchy" : [ ]
//	 }
//	 {
//		"_id" : 2,
//		"name" : "Eliot",
//		"reportsTo" : "Dev",
//		"reportingHierarchy" : [
//		   { "_id" : 1, "name" : "Dev" }
//		]
//	 }
//	 {
//		"_id" : 3,
//		"name" : "Ron",
//		"reportsTo" : "Eliot",
//		"reportingHierarchy" : [
//		   { "_id" : 1, "name" : "Dev" },
//		   { "_id" : 2, "name" : "Eliot", "reportsTo" : "Dev" }
//		]
//	 }
//	 {
//		"_id" : 4,
//		"name" : "Andrew",
//		"reportsTo" : "Eliot",
//		"reportingHierarchy" : [
//		   { "_id" : 1, "name" : "Dev" },
//		   { "_id" : 2, "name" : "Eliot", "reportsTo" : "Dev" }
//		]
//	 }

const (
	databaseName = "familyTree"

	relativesFieldName = "relatives"
	depthFieldName     = "depthField"

	personCollectionName = "person"
)

// type Relative struct {
// 	Person
// 	DepthField int64 `bson:"depthField,omitempty"`
// }

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

func buildPersonFromPersonWithRelatives(ctx context.Context, personWithRelatives PersonWithRelatives) (domain.Person, error) {
	personRelativesMap := make(map[string]*domain.Person, len(personWithRelatives.Relatives)+1)
	person := domain.Person{
		ID:     personWithRelatives.ID.Hex(),
		Name:   personWithRelatives.Name,
		Gender: domain.GenderType(personWithRelatives.Gender),
	}

	for _, parent := range personWithRelatives.Parents {
		person.Parents = append(person.Parents, domain.Person{ID: parent.Hex()})
	}

	for _, children := range personWithRelatives.Children {
		person.Children = append(person.Children, domain.Person{ID: children.Hex()})
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
				currentPerson.Parents[i] = *parent
			}
		}

		for i, emptyChildren := range currentPerson.Children {
			children := personRelativesMap[emptyChildren.ID]
			if children != nil {
				currentPerson.Children[i] = *children
			}
		}
	}

	return person, nil

}

func (pr PersonRepository) GetPersonFamilyTreeByID(ctx context.Context, personID string, maxDepth *int64) (*domain.Person, error) {
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

	fmt.Printf("%+v\n", personRelatives.Name)
	personAscendants, err := buildPersonFromPersonWithRelatives(ctx, personRelatives)
	if err != nil {
		return nil, err
	}

	fmt.Print("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	fmt.Printf("%+v\n", personAscendants) // TODO: REMOVE THIS

	return &personAscendants, nil
}

func (pr PersonRepository) GetDescendantsByPersonID(ctx context.Context, personID string, maxDepth *int64) ([]domain.Person, error) {
	return nil, nil
}

// func (pr PersonRepository) GetFamilyTreeByPersonID(ctx context.Context, personID string) (*domain.Person, error) {
// 	// pegar todos os ascendentes, não precisa de TODOS os relationships
// 	// AS RELAÇÕES SÃO SÓ PARA A PESSOA PROCURADA, POR EXEMPLO QND PEGAMOS JAQUELINE QUE É UMA IRMÃ, NÃO PEGAMOS ERIC QUE É O PAI DELA. Não precisamos de todas as relações de todos os nós.
// 	// Nao ter informação redundante ou seja, Pheobe é filha de marcos e Marcos é pai de Phoebe
// 	return nil, nil
// }

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
