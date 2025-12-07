//go:build wireinject
// +build wireinject

package cashtrack

import (
	"context"
	"net/http"
)
import "github.com/google/wire"

func handlers() []*Handler {
	return []*Handler{NewGreetHandler(), NewTodoHandler()}
}

func InitializeHttpServer(ctx context.Context) (*http.Server, error) {
	wire.Build(
		handlers,
		ProvideConfig,
		wire.FieldsOf(new(Config), "ServerConfig"),
		NewHttpServer,
	)
	return nil, nil
}
