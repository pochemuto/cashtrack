//go:build wireinject
// +build wireinject

package cashtrack

import (
	"context"
	"net/http"
)
import "github.com/google/wire"

func handlers(todo *TodoHandler, greet *GreetHandler) []*Handler {
	return []*Handler{(*Handler)(todo), (*Handler)(greet)}
}

func InitializeHttpServer(ctx context.Context) (*http.Server, error) {
	wire.Build(
		handlers,
		NewGreetHandler, NewTodoHandler,
		ProvideConfig,
		wire.FieldsOf(new(Config), "ServerConfig", "db"),
		NewHttpServer, NewPgxPool, NewDB,
	)
	return nil, nil
}
