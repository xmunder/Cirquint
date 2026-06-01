package identity

import "context"

type Actor struct {
	ID string `json:"id"`
}

type actorKey struct{}

func ContextWithActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

func ActorFromContext(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorKey{}).(Actor)
	return actor
}
