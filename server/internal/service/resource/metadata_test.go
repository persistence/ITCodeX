package resource

import (
	"context"
	"errors"
	"testing"

	"itcodex/server/internal/service/metadata"
)

func TestMetadataCRUDHandlers(t *testing.T) {
	repository := &fakeRepository{}
	var resolved string
	resolver := ResolverFunc(func(_ context.Context, resourceName string) (Repository, error) {
		resolved = resourceName
		return repository, nil
	})
	manager := NewManager()
	unregister := manager.RegisterMetadataCRUD(resolver)

	dispatch := func(action string, params map[string]any, data, options any) any {
		t.Helper()
		result, err := manager.Dispatch(context.Background(), &ActionRequest{
			Resource: "users",
			Action:   action,
			Params:   params,
			Data:     data,
			Options:  options,
		})
		if err != nil {
			t.Fatalf("%s: %v", action, err)
		}
		return result
	}

	list := dispatch(ActionList, map[string]any{"filter": map[string]any{"active": true}}, nil, nil)
	if result, ok := list.(CollectionListResult); !ok || result.Total != 1 || result.List[0]["name"] != "Alice" {
		t.Fatalf("unexpected list result: %#v", list)
	}
	if repository.findOptions.Filter["active"] != true {
		t.Fatalf("list filter was not forwarded: %#v", repository.findOptions.Filter)
	}

	dispatch(ActionGet, map[string]any{"id": int64(7)}, nil, nil)
	if repository.findOneOptions.FilterByTk != int64(7) {
		t.Fatalf("get id was not forwarded: %#v", repository.findOneOptions.FilterByTk)
	}

	if count := dispatch(ActionCount, nil, nil, nil); count != 3 {
		t.Fatalf("unexpected count: %#v", count)
	}

	dispatch(ActionCreate, nil, map[string]any{"name": "Bob"}, map[string]any{"whitelist": []string{"name"}})
	if repository.createOptions.Values["name"] != "Bob" || len(repository.createOptions.Whitelist) != 1 {
		t.Fatalf("create input was not forwarded: %#v", repository.createOptions)
	}

	dispatch(ActionCreateMany, nil, []map[string]any{{"name": "A"}, {"name": "B"}}, nil)
	if len(repository.createManyOptions.Records) != 2 {
		t.Fatalf("createMany records were not forwarded: %#v", repository.createManyOptions)
	}

	dispatch(ActionUpdate, map[string]any{"id": 9}, map[string]any{"name": "Carol"}, nil)
	if repository.updateOptions.FilterByTk != float64(9) && repository.updateOptions.FilterByTk != 9 {
		t.Fatalf("update id was not forwarded: %#v", repository.updateOptions.FilterByTk)
	}
	if repository.updateOptions.Values["name"] != "Carol" {
		t.Fatalf("update values were not forwarded: %#v", repository.updateOptions.Values)
	}

	dispatch(ActionUpdateMany, map[string]any{"filter": map[string]any{"active": false}}, map[string]any{"active": true}, nil)
	if repository.updateOptions.Filter["active"] != false {
		t.Fatalf("updateMany filter was not forwarded: %#v", repository.updateOptions.Filter)
	}

	dispatch(ActionDestroy, map[string]any{"id": "u-1"}, nil, nil)
	if repository.destroyOptions.FilterByTk != "u-1" {
		t.Fatalf("destroy id was not forwarded: %#v", repository.destroyOptions.FilterByTk)
	}
	dispatch(ActionDestroyMany, map[string]any{"truncate": true}, nil, nil)
	if !repository.destroyOptions.Truncate {
		t.Fatal("destroyMany truncate was not forwarded")
	}

	dispatch(ActionAssociationList, map[string]any{"id": 1, "association": "roles"}, nil, nil)
	dispatch(ActionAssociationAdd, map[string]any{"id": 1, "association": "roles"}, map[string]any{"id": 2}, nil)
	dispatch(ActionAssociationSet, map[string]any{"id": 1, "association": "roles"}, []any{2, 3}, nil)
	dispatch(ActionAssociationRemove, map[string]any{"id": 1, "association": "roles"}, map[string]any{"id": 2}, nil)
	if repository.association != "roles" || repository.associationCalls != 4 {
		t.Fatalf("association actions were not forwarded: %q, %d", repository.association, repository.associationCalls)
	}
	if resolved != "users" {
		t.Fatalf("resolved %q, want users", resolved)
	}

	unregister()
	_, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: ActionList})
	if !errors.Is(err, ErrActionNotFound) {
		t.Fatalf("got %v after unregister, want ErrActionNotFound", err)
	}
}

func TestMetadataResolverErrors(t *testing.T) {
	manager := NewManager()
	manager.RegisterMetadataCRUD(ResolverFunc(func(context.Context, string) (Repository, error) {
		return nil, ErrRepositoryNotFound
	}))
	_, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "missing", Action: ActionList})
	if !errors.Is(err, ErrRepositoryNotFound) {
		t.Fatalf("got %v, want ErrRepositoryNotFound", err)
	}
}

type fakeRepository struct {
	findOptions       *metadata.FindOptions
	findOneOptions    *metadata.FindOneOptions
	createOptions     *metadata.CreateOptions
	createManyOptions *metadata.CreateManyOptions
	updateOptions     *metadata.UpdateOptions
	destroyOptions    *metadata.DestroyOptions
	association       string
	associationCalls  int
}

func (r *fakeRepository) FindAndCount(_ context.Context, opts *metadata.FindOptions) ([]*metadata.Record, int, error) {
	r.findOptions = opts
	return []*metadata.Record{metadata.NewRecord(map[string]any{"name": "Alice"})}, 1, nil
}

func (r *fakeRepository) FindOne(_ context.Context, opts *metadata.FindOneOptions) (*metadata.Record, error) {
	r.findOneOptions = opts
	return metadata.NewRecord(map[string]any{"id": opts.FilterByTk}), nil
}

func (*fakeRepository) Count(context.Context, *metadata.CountOptions) (int, error) {
	return 3, nil
}

func (r *fakeRepository) Create(_ context.Context, opts *metadata.CreateOptions) (*metadata.Record, error) {
	r.createOptions = opts
	return metadata.NewRecord(opts.Values), nil
}

func (r *fakeRepository) CreateMany(_ context.Context, opts *metadata.CreateManyOptions) ([]*metadata.Record, error) {
	r.createManyOptions = opts
	result := make([]*metadata.Record, 0, len(opts.Records))
	for _, values := range opts.Records {
		result = append(result, metadata.NewRecord(values))
	}
	return result, nil
}

func (r *fakeRepository) Update(_ context.Context, opts *metadata.UpdateOptions) (*metadata.Record, int, error) {
	r.updateOptions = opts
	return metadata.NewRecord(opts.Values), 1, nil
}

func (r *fakeRepository) Destroy(_ context.Context, opts *metadata.DestroyOptions) (int, error) {
	r.destroyOptions = opts
	return 1, nil
}

func (r *fakeRepository) ListAssociation(_ context.Context, _ any, association string) ([]map[string]any, error) {
	r.association = association
	r.associationCalls++
	return []map[string]any{{"id": 2}}, nil
}

func (r *fakeRepository) AddAssociation(_ context.Context, _ any, association string, _ any) error {
	r.association = association
	r.associationCalls++
	return nil
}

func (r *fakeRepository) SetAssociation(_ context.Context, _ any, association string, _ any) error {
	r.association = association
	r.associationCalls++
	return nil
}

func (r *fakeRepository) RemoveAssociation(_ context.Context, _ any, association string, _ any) error {
	r.association = association
	r.associationCalls++
	return nil
}
