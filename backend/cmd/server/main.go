package main

import (
	cashtrack "cashtrack/backend"
	"context"
)

func main() {
	server, err := cashtrack.InitializeHttpServer(context.Background())
	if err != nil {
		panic(err)
	}

	err = server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
