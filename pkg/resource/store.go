package resource

import (
	"context"
	"errors"
)

var (
	ErrNotFound = errors.New("resource: not found")
	ErrConflict = errors.New("resource: conflict")
)

type Store interface {
	CreateResourceClass(ctx context.Context, rc ResourceClass) (ResourceClass, error)
	GetResourceClass(ctx context.Context, id int64) (ResourceClass, error)
	ListResourceClasses(ctx context.Context) ([]ResourceClass, error)
	UpdateResourceClass(ctx context.Context, rc ResourceClass) error
	DeleteResourceClass(ctx context.Context, id int64) error

	CreateResourceSet(ctx context.Context, rs ResourceSet) (ResourceSet, error)
	GetResourceSet(ctx context.Context, id int64) (ResourceSet, error)
	ListResourceSetsByCourse(ctx context.Context, courseID int64) ([]ResourceSet, error)
	ListResourceSetsByClass(ctx context.Context, resourceClassID int64) ([]ResourceSet, error)
	UpdateResourceSet(ctx context.Context, rs ResourceSet) error
	DeleteResourceSet(ctx context.Context, id int64) error

	CreateResource(ctx context.Context, r Resource) (Resource, error)
	GetResource(ctx context.Context, id int64) (Resource, error)
	ListResourcesBySet(ctx context.Context, resourceSetID int64) ([]Resource, error)
	UpdateResource(ctx context.Context, r Resource) error
	DeleteResource(ctx context.Context, id int64) error
}
