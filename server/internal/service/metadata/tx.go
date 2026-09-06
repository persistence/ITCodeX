package metadata

import (
	"context"
	"fmt"
)

type writeUnitKey struct{}
type afterCommitRunningKey struct{}

type writeUnit struct {
	tx    *txDB
	after []func()
}

func ContextWithWriteUnit(ctx context.Context, u *writeUnit) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, writeUnitKey{}, u)
}

func WriteUnitFromContext(ctx context.Context) *writeUnit {
	if ctx == nil {
		return nil
	}
	u, _ := ctx.Value(writeUnitKey{}).(*writeUnit)
	return u
}

func withAfterCommitRunning(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, afterCommitRunningKey{}, true)
}

func afterCommitRunning(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	v, _ := ctx.Value(afterCommitRunningKey{}).(bool)
	return v
}

func (r *GenericRepository) inWriteUnit(ctx context.Context) bool {
	return WriteUnitFromContext(ctx) != nil || r.unit != nil || r.tx != nil
}

func (r *GenericRepository) withUnit(u *writeUnit) *GenericRepository {
	return &GenericRepository{
		coll: r.coll,
		tx:   u.tx.tx,
		txdb: u.tx,
		unit: u,
	}
}

func (r *GenericRepository) registerAfterCommit(ctx context.Context, fn func()) {
	if fn == nil || afterCommitRunning(ctx) {
		return
	}
	if u := WriteUnitFromContext(ctx); u != nil {
		u.after = append(u.after, fn)
		return
	}
	if r.unit != nil {
		r.unit.after = append(r.unit.after, fn)
	}
}

func (r *GenericRepository) fireAfterCommit(hooks []func()) {
	for _, h := range hooks {
		h()
	}
}

func (r *GenericRepository) runWrite(ctx context.Context, fn func(ctx context.Context, repo *GenericRepository) error) error {
	if u := WriteUnitFromContext(ctx); u != nil {
		return fn(ctx, r.withUnit(u))
	}
	if r.unit != nil {
		ctx = ContextWithWriteUnit(ctx, r.unit)
		return fn(ctx, r)
	}
	if r.tx != nil {
		u := r.unit
		if u == nil {
			u = &writeUnit{tx: &txDB{tx: r.tx}}
		}
		ctx = ContextWithWriteUnit(ctx, u)
		return fn(ctx, r.withUnit(u))
	}

	sqlDB := r.coll.Db().SqlDB()
	if sqlDB == nil {
		return fmt.Errorf("no sql db available")
	}
	tx, err := sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return NewSystemError(err)
	}
	u := &writeUnit{tx: &txDB{tx: tx}}
	ctx = ContextWithWriteUnit(ctx, u)
	if err := fn(ctx, r.withUnit(u)); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return NewSystemError(err)
	}
	r.fireAfterCommit(u.after)
	return nil
}
