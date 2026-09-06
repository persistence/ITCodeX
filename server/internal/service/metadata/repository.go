package metadata

import "context"

type Repository interface {
	Find(ctx context.Context, opts *FindOptions) ([]*Record, error)
	FindOne(ctx context.Context, opts *FindOneOptions) (*Record, error)
	FindAndCount(ctx context.Context, opts *FindOptions) ([]*Record, int, error)
	Count(ctx context.Context, opts *CountOptions) (int, error)
	Create(ctx context.Context, opts *CreateOptions) (*Record, error)
	CreateMany(ctx context.Context, opts *CreateManyOptions) ([]*Record, error)
	Update(ctx context.Context, opts *UpdateOptions) (*Record, int, error)
	Destroy(ctx context.Context, opts *DestroyOptions) (int, error)
	Transaction(ctx context.Context, fn func(tx Repository) error) error
	Collection() *Collection
	ListAssociation(ctx context.Context, sourceID any, association string) ([]map[string]any, error)
	AddAssociation(ctx context.Context, sourceID any, association string, body any) error
	SetAssociation(ctx context.Context, sourceID any, association string, body any) error
	RemoveAssociation(ctx context.Context, sourceID any, association string, body any) error
}
