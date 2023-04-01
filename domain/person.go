package domain

type RelationshipType string

const (
	ParentRelashionship  RelationshipType = "parent"
	ChildRelashionship   RelationshipType = "child"
	SiblingRelashionship RelationshipType = "sibling"
	NephewRelashionship  RelationshipType = "nephew"
	CousinRelashionship  RelationshipType = "cousin"
)

type GenderType string

const (
	Male   GenderType = "male"
	Female GenderType = "female"
)

func (s GenderType) IsValid() bool {
	switch s {
	case Female, Male:
		return true
	}

	return false
}

type RelationshipPerson struct {
	ID     string
	Name   string
	Gender GenderType
}

type Relationship struct {
	Person       RelationshipPerson
	Relationship RelationshipType
}

// type Person struct {
// 	Person

// 	BaconsNumber int64
// }

// Bread first search começando com Phoebe.
// Visito as crianças, visito o pai
// Vou adicionando geração quando pesquiso para o PAI e subtraindo qnd buscando pelos filhos.
// adiciono visited nodes
// crio um map com geração! e faço, geração do current node +1 -1 para pegar relationships

type Person struct {
	ID       string
	Name     string
	Gender   GenderType
	Parents  []Person
	Children []Person

	// Ascendants    map[int][]Person
	// Descendants   map[int][]Person
	Relationships []Relationship
}

func buildRelationshipFromPerson(person Person, relationshipType RelationshipType) Relationship {
	return Relationship{
		Person: RelationshipPerson{
			ID:     person.ID,
			Name:   person.Name,
			Gender: person.Gender,
		},
		Relationship: relationshipType,
	}
}

// salvar pessoa num array e após 3 pulos remover. Usar um breadfirst search para buscar as relações.
// func (pr *Person) BuildTreeFromAscendantsAndParentsDescendants() error {
// 	// com ascendant eu vou achar parents /
// 	//[mae, pai, paiAvo, paiAvó, mãoAvo, mãeAvó]

// 	// min := int(math.Min(float64(len(pr.Ascendants)), 2))
// 	// for i := 0; i < min; i++ {
// 	// 	ascendant := pr.Ascendants[i]

// 	// 	if ascendant.BaconsNumber == 1 {
// 	// 		pr.Relationships = append(pr.Relationships, Relationship{
// 	// 			Person: RelationshipPerson{
// 	// 				ID:     ascendant.ID,
// 	// 				Name:   ascendant.Name,
// 	// 				Gender: ascendant.Gender,
// 	// 			},
// 	// 			Relationship: Parent,
// 	// 		})
// 	// 	}
// 	// }

// 	// fazer a mesma coisa com MOTHER

// 	// não precisa desse aqui?
// 	if len(pr.Ascendants) > 0 {
// 		for _, parent := range pr.Ascendants[1] {
// 			// pr.Relationships = append(pr.Relationships, buildRelationshipFromPerson(parent, ParentRelashionship))
// 			if parent.Gender == Male {
// 				pr.Father = &parent
// 			} else {
// 				pr.Mother = &parent
// 			}

// 			parent.Children = append(parent.Children, *pr)
// 		}
// 	}

// 	for i := 1; i < len(pr.Ascendants); i++ {
// 		children := pr.Ascendants[i]

// 		if len(pr.Ascendants) < i+1 {
// 			break
// 		}

// 		for _, currentChildren := range children {
// 			possibleParents := pr.Ascendants[i+1]

// 			amountOfParentsToFind := 0
// 			if currentChildren.Father != nil {
// 				amountOfParentsToFind++
// 			}

// 			if currentChildren.Mother != nil {
// 				amountOfParentsToFind++
// 			}

// 			for _, possibleParent := range possibleParents {
// 				if (currentChildren.Father != nil && possibleParent.ID == currentChildren.Father.ID) ||
// 					(currentChildren.Mother != nil && possibleParent.ID == currentChildren.Mother.ID) {
// 					// currentChildren.Relationships = append(currentChildren.Relationships, buildRelationshipFromPerson(possibleParent, ParentRelashionship))
// 					if possibleParent.Gender == Male {
// 						currentChildren.Father = &possibleParent
// 					} else {
// 						currentChildren.Mother = &possibleParent
// 					}

// 					possibleParent.Children = append(possibleParent.Children, currentChildren)
// 					amountOfParentsToFind--
// 				}

// 				if amountOfParentsToFind == 0 {
// 					break
// 				}
// 			}
// 		}
// 	}

// 	if pr.Father != nil && len(pr.Father.Descendants) > 0 {
// 		for _, children := range pr.Father.Descendants[1] {
// 			children.Father = pr.Father
// 			pr.Father.Children = append(pr.Father.Children, children)
// 		}

// 		//NOTE: Using father and mother descendants to get brothers and nephew
// 		for i := 1; i < len(pr.Father.Descendants); i++ {
// 			parents := pr.Father.Descendants[i]

// 			if len(pr.Ascendants) < i+1 {
// 				break
// 			}

// 			for _, currentParent := range parents {
// 				// current Parent, Caio
// 				possibleChildren := pr.Ascendants[i+1]

// 				amountOfChildrenToFind := len(currentParent.Children)

// 				for _, currentPossibleChildren := range possibleChildren {
// 					// Cuaua
// 					if (currentParent.Father != nil && currentPossibleChildren.ID == currentParent.Father.ID) ||
// 						(currentParent.Mother != nil && currentPossibleChildren.ID == currentParent.Mother.ID) {
// 						// currentParent.Relationships = append(currentParent.Relationships, buildRelationshipFromPerson(currentPossibleChildren, ParentRelashionship))
// 						if currentPossibleChildren.Gender == Male {
// 							currentParent.Father = &currentPossibleChildren
// 						} else {
// 							currentParent.Mother = &currentPossibleChildren
// 						}

// 						currentPossibleChildren.Children = append(currentPossibleChildren.Children, currentParent)
// 						amountOfChildrenToFind--
// 					}

// 					if amountOfChildrenToFind == 0 {
// 						break
// 					}
// 				}
// 			}

// 		}
// 	}

// 	return nil
// }
