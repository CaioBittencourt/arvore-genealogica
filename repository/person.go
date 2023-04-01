package repository

import (
	"context"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
)

type PersonRepository interface {
	GetPersonFamilyTreeByID(ctx context.Context, personID string, maxDepth *int64) (*domain.FamilyTree, error)
	Store(ctx context.Context, person domain.Person) (*domain.Person, error)
}
