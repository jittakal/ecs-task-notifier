package main

import (
	"context"
	"log"
	"testing"
)

func TestHandler(t *testing.T) {
	ctx := context.TODO()

	err := HandleRequest(ctx, nil)

	if err != nil {
		log.Fatal(err)
	}
}
