package resource

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"itcodex/server/internal/service/metadata"
)

const (
	ActionList              = "list"
	ActionGet               = "get"
	ActionCount             = "count"
	ActionCreate            = "create"
	ActionCreateMany        = "createMany"
	ActionUpdate            = "update"
	ActionUpdateMany        = "updateMany"
	ActionDestroy           = "destroy"
	ActionDestroyMany       = "destroyMany"
	ActionAssociationList   = "association:list"
	ActionAssociationAdd    = "association:add"
	ActionAssociationSet    = "association:set"
	ActionAssociationRemove = "association:remove"
	ActionUpload            = "upload"
	ActionFileGet           = "file:get"
)

var ErrRepositoryNotFound = errors.New("resource: metadata repository not found")

// Repository is the part of metadata.Repository used by collection actions.
// metadata.Repository satisfies this interface directly.
type Repository interface {
	FindAndCount(context.Context, *metadata.FindOptions) ([]*metadata.Record, int, error)
	FindOne(context.Context, *metadata.FindOneOptions) (*metadata.Record, error)
	Count(context.Context, *metadata.CountOptions) (int, error)
	Create(context.Context, *metadata.CreateOptions) (*metadata.Record, error)
	CreateMany(context.Context, *metadata.CreateManyOptions) ([]*metadata.Record, error)
	Update(context.Context, *metadata.UpdateOptions) (*metadata.Record, int, error)
	Destroy(context.Context, *metadata.DestroyOptions) (int, error)
	ListAssociation(context.Context, any, string) ([]map[string]any, error)
	AddAssociation(context.Context, any, string, any) error
	SetAssociation(context.Context, any, string, any) error
	RemoveAssociation(context.Context, any, string, any) error
}

// Resolver resolves a dynamic resource name to its collection repository.
type Resolver interface {
	Resolve(context.Context, string) (Repository, error)
}

type ResolverFunc func(context.Context, string) (Repository, error)

func (f ResolverFunc) Resolve(ctx context.Context, resourceName string) (Repository, error) {
	return f(ctx, resourceName)
}

// DatabaseResolver adapts metadata.Database to Resolver.
type DatabaseResolver struct {
	Database *metadata.Database
}

func (r DatabaseResolver) Resolve(_ context.Context, resourceName string) (Repository, error) {
	if r.Database == nil {
		return nil, ErrRepositoryNotFound
	}
	collection := r.Database.Collection(resourceName)
	if collection == nil {
		return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, resourceName)
	}
	return collection.Repository(), nil
}

// CollectionListResult is returned by the list action.
type CollectionListResult struct {
	List  []map[string]any
	Total int
}

// MutationResult is returned by update and destroy actions.
type MutationResult struct {
	Record   map[string]any
	Affected int
}

// RegisterMetadataCRUD registers all built-in collection actions. The optional
// pattern defaults to "*" and can restrict which resources use the resolver.
func (m *Manager) RegisterMetadataCRUD(resolver Resolver, resourcePattern ...string) func() {
	if m == nil || resolver == nil {
		return func() {}
	}
	pattern := "*"
	if len(resourcePattern) > 0 && resourcePattern[0] != "" {
		pattern = resourcePattern[0]
	}

	handler := metadataActionHandler(resolver)
	actions := []string{
		ActionList,
		ActionGet,
		ActionCount,
		ActionCreate,
		ActionCreateMany,
		ActionUpdate,
		ActionUpdateMany,
		ActionDestroy,
		ActionDestroyMany,
		ActionAssociationList,
		ActionAssociationAdd,
		ActionAssociationSet,
		ActionAssociationRemove,
	}
	unregister := make([]func(), 0, len(actions))
	for _, action := range actions {
		unregister = append(unregister, m.Register(pattern, action, handler))
	}
	return func() {
		for _, remove := range unregister {
			remove()
		}
	}
}

func RegisterMetadataCRUD(manager *Manager, resolver Resolver, resourcePattern ...string) func() {
	if manager == nil {
		return func() {}
	}
	return manager.RegisterMetadataCRUD(resolver, resourcePattern...)
}

// RegisterCollectionHandlers is an alias with a more domain-specific name.
func RegisterCollectionHandlers(manager *Manager, resolver Resolver, resourcePattern ...string) func() {
	return RegisterMetadataCRUD(manager, resolver, resourcePattern...)
}

func metadataActionHandler(resolver Resolver) ActionHandler {
	return func(actionContext *ActionContext) (any, error) {
		request := actionContext.Request
		repository, err := resolver.Resolve(actionContext.Context, request.Resource)
		if err != nil {
			return nil, err
		}
		if repository == nil {
			return nil, fmt.Errorf("%w: %s", ErrRepositoryNotFound, request.Resource)
		}

		switch request.Action {
		case ActionList:
			opts := &metadata.FindOptions{}
			if err := decodeRequestOptions(request, opts); err != nil {
				return nil, err
			}
			records, total, err := repository.FindAndCount(actionContext.Context, opts)
			if err != nil {
				return nil, err
			}
			return CollectionListResult{List: recordMaps(records), Total: total}, nil

		case ActionGet:
			opts := &metadata.FindOneOptions{}
			if err := decodeRequestOptions(request, opts); err != nil {
				return nil, err
			}
			if id, ok := request.Params["id"]; ok {
				opts.FilterByTk = id
			}
			record, err := repository.FindOne(actionContext.Context, opts)
			if err != nil || record == nil {
				return nil, err
			}
			return record.Data(), nil

		case ActionCount:
			opts := &metadata.CountOptions{}
			if err := decodeRequestOptions(request, opts); err != nil {
				return nil, err
			}
			return repository.Count(actionContext.Context, opts)

		case ActionCreate:
			opts := &metadata.CreateOptions{}
			if typed, ok := request.Data.(*metadata.CreateOptions); ok {
				opts = typed
			} else {
				if err := decodeInto(request.Options, opts); err != nil {
					return nil, err
				}
				values, err := mapData(request.Data)
				if err != nil {
					return nil, err
				}
				opts.Values = values
			}
			record, err := repository.Create(actionContext.Context, opts)
			if err != nil || record == nil {
				return nil, err
			}
			return record.Data(), nil

		case ActionCreateMany:
			opts := &metadata.CreateManyOptions{}
			if typed, ok := request.Data.(*metadata.CreateManyOptions); ok {
				opts = typed
			} else {
				if err := decodeInto(request.Options, opts); err != nil {
					return nil, err
				}
				records, err := mapSliceData(request.Data)
				if err != nil {
					return nil, err
				}
				opts.Records = records
			}
			records, err := repository.CreateMany(actionContext.Context, opts)
			if err != nil {
				return nil, err
			}
			return recordMaps(records), nil

		case ActionUpdate, ActionUpdateMany:
			opts := &metadata.UpdateOptions{}
			if typed, ok := request.Data.(*metadata.UpdateOptions); ok {
				opts = typed
			} else {
				if err := decodeRequestOptions(request, opts); err != nil {
					return nil, err
				}
				values, err := mapData(request.Data)
				if err != nil {
					return nil, err
				}
				opts.Values = values
			}
			if request.Action == ActionUpdate {
				if id, ok := request.Params["id"]; ok {
					opts.FilterByTk = id
				}
			}
			record, affected, err := repository.Update(actionContext.Context, opts)
			if err != nil {
				return nil, err
			}
			return MutationResult{Record: recordMap(record), Affected: affected}, nil

		case ActionDestroy, ActionDestroyMany:
			opts := &metadata.DestroyOptions{}
			if err := decodeRequestOptions(request, opts); err != nil {
				return nil, err
			}
			if request.Action == ActionDestroy {
				if id, ok := request.Params["id"]; ok {
					opts.FilterByTk = id
				}
			}
			affected, err := repository.Destroy(actionContext.Context, opts)
			if err != nil {
				return nil, err
			}
			return MutationResult{Affected: affected}, nil

		case ActionAssociationList:
			id, association := associationParams(request)
			return repository.ListAssociation(actionContext.Context, id, association)
		case ActionAssociationAdd:
			id, association := associationParams(request)
			return nil, repository.AddAssociation(actionContext.Context, id, association, request.Data)
		case ActionAssociationSet:
			id, association := associationParams(request)
			return nil, repository.SetAssociation(actionContext.Context, id, association, request.Data)
		case ActionAssociationRemove:
			id, association := associationParams(request)
			return nil, repository.RemoveAssociation(actionContext.Context, id, association, request.Data)
		default:
			return nil, fmt.Errorf("%w: %s/%s", ErrActionNotFound, request.Resource, request.Action)
		}
	}
}

func decodeRequestOptions(request *ActionRequest, destination any) error {
	if err := decodeInto(request.Params, destination); err != nil {
		return err
	}
	return decodeInto(request.Options, destination)
}

func decodeInto(value, destination any) error {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("resource: encode options: %w", err)
	}
	if err := json.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("resource: decode options: %w", err)
	}
	return nil
}

func mapData(value any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	if values, ok := value.(map[string]any); ok {
		return values, nil
	}
	var values map[string]any
	if err := decodeInto(value, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func mapSliceData(value any) ([]map[string]any, error) {
	if value == nil {
		return []map[string]any{}, nil
	}
	if records, ok := value.([]map[string]any); ok {
		return records, nil
	}
	var records []map[string]any
	if err := decodeInto(value, &records); err != nil {
		return nil, err
	}
	return records, nil
}

func recordMaps(records []*metadata.Record) []map[string]any {
	result := make([]map[string]any, 0, len(records))
	for _, record := range records {
		if record != nil {
			result = append(result, record.Data())
		}
	}
	return result
}

func recordMap(record *metadata.Record) map[string]any {
	if record == nil {
		return nil
	}
	return record.Data()
}

func associationParams(request *ActionRequest) (any, string) {
	return request.Params["id"], fmt.Sprint(request.Params["association"])
}
