package entity

import (
	"context"
)

// contextKey is a private type for context keys to avoid collisions
type contextKey int

const (
	// entityContextKey is the key for storing an entity.Context in a context.Context
	entityContextKey contextKey = iota
)

// ContextWithEntityID adds an EntityID to a context.Context.
func ContextWithEntityID(ctx context.Context, entityID EntityID) context.Context {
	return context.WithValue(ctx, entityContextKey, Context{EntityID: entityID})
}

// ContextWithEntity adds a full entity.Context to a context.Context.
func ContextWithEntity(ctx context.Context, entityCtx Context) context.Context {
	return context.WithValue(ctx, entityContextKey, entityCtx)
}

// GetEntityContext retrieves the entity.Context from a context.Context.
// If no entity.Context is found, it returns a zero-valued entity.Context and false.
func GetEntityContext(ctx context.Context) (Context, bool) {
	entityCtx, ok := ctx.Value(entityContextKey).(Context)
	return entityCtx, ok
}

// MustGetEntityContext retrieves the entity.Context from a context.Context.
// Panics if no entity.Context is found, so only use when you are sure a Context exists.
func MustGetEntityContext(ctx context.Context) Context {
	entityCtx, ok := GetEntityContext(ctx)
	if !ok {
		panic("entity.Context not found in context.Context")
	}
	return entityCtx
}
