package repository

import (
	"context"

	"github.com/CaioBittencourt/arvore-genealogica/domain"
)

type PersonRepository interface {
	GetPersonFamilyGraphByID(ctx context.Context, personID string, maxDepth *int64) (*domain.FamilyGraph, error)
	Store(ctx context.Context, person domain.Person) (*domain.Person, error)
}
