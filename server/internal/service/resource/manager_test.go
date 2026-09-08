package resource

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
)

func TestManagerDispatchAndSpecificity(t *testing.T) {
	manager := NewManager()
	manager.Register("*", "*", resultHandler("fallback"))
	manager.Register("users*", "get", resultHandler("resource wildcard"))
	manager.Register("users", "get", resultHandler("exact"))

	got, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: "get"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "exact" {
		t.Fatalf("got %q, want exact", got)
	}

	got, err = manager.Dispatch(context.Background(), &ActionRequest{Resource: "users_archive", Action: "get"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "resource wildcard" {
		t.Fatalf("got %q, want resource wildcard", got)
	}
}

func TestManagerMiddlewareOrderAndState(t *testing.T) {
	manager := NewManager()
	var calls []string
	manager.Use(
		func(next ActionHandler) ActionHandler {
			return func(ctx *ActionContext) (any, error) {
				calls = append(calls, "first:before")
				ctx.State["value"] = "ok"
				result, err := next(ctx)
				calls = append(calls, "first:after")
				return result, err
			}
		},
		func(next ActionHandler) ActionHandler {
			return func(ctx *ActionContext) (any, error) {
				calls = append(calls, "second:before")
				result, err := next(ctx)
				calls = append(calls, "second:after")
				return result, err
			}
		},
	)
	manager.Register("users", "get", func(ctx *ActionContext) (any, error) {
		calls = append(calls, "handler")
		return ctx.State["value"], nil
	})

	got, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: "get"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "ok" {
		t.Fatalf("got %v, want ok", got)
	}
	want := "[first:before second:before handler second:after first:after]"
	if fmt.Sprint(calls) != want {
		t.Fatalf("calls %v, want %s", calls, want)
	}
}

func TestManagerRegisterReplaceAndUnregister(t *testing.T) {
	manager := NewManager()
	removeOld := manager.Register("users", "get", resultHandler("old"))
	removeNew := manager.Register("users", "get", resultHandler("new"))

	removeOld()
	got, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: "get"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "new" {
		t.Fatalf("got %v, want new", got)
	}

	removeNew()
	_, err = manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: "get"})
	if !errors.Is(err, ErrActionNotFound) {
		t.Fatalf("got %v, want ErrActionNotFound", err)
	}
}

func TestManagerConcurrentAccess(t *testing.T) {
	manager := NewManager()
	manager.Register("*", "*", resultHandler("ok"))

	var failures atomic.Int32
	var wait sync.WaitGroup
	for i := 0; i < 20; i++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			remove := manager.Register(fmt.Sprintf("temporary-%d", index), "get", resultHandler(index))
			if _, err := manager.Dispatch(context.Background(), &ActionRequest{Resource: "users", Action: "list"}); err != nil {
				failures.Add(1)
			}
			remove()
		}(i)
	}
	wait.Wait()
	if failures.Load() != 0 {
		t.Fatalf("%d concurrent dispatches failed", failures.Load())
	}
}

func TestManagerRejectsInvalidRequest(t *testing.T) {
	manager := NewManager()
	if _, err := manager.Dispatch(context.Background(), nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("got %v, want ErrInvalidRequest", err)
	}
}

func resultHandler(value any) ActionHandler {
	return func(*ActionContext) (any, error) {
		return value, nil
	}
}
