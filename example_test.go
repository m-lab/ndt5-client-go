package ndt5_test

import (
	"context"
	"log"
	"time"

	"github.com/bassosimone/ndt5-client-go"
)

// This shows how to run a ndt5 test.
func Example() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	client := ndt5.NewClient("ndt5-client-go-example", "0.1.0")
	ch, err := client.Start(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for ev := range ch {
		log.Printf("%+v", ev)
	}
}
