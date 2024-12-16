package middleware_test

import (
	"context"
	"time"

	"golang.org/x/time/rate"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"

	"github.com/gotd/contrib/middleware/floodwait"
	"github.com/gotd/contrib/middleware/ratelimit"
)

func Example() {
	// Create a new telegram.Client instance that handles FLOOD_WAIT errors
	// and limits request rate to 10 Hz with max burst size of 5 request.
	//
	// Note that you must not use test app credentials in production.
	// See https://core.telegram.org/api/obtaining_api_id
	client := telegram.NewClient(
		telegram.TestAppID,
		telegram.TestAppHash,
		telegram.Options{
			Middlewares: []telegram.Middleware{
				floodwait.NewSimpleWaiter().WithMaxRetries(10),
				ratelimit.New(rate.Every(100*time.Millisecond), 5),
			},
		},
	)

	api := tg.NewClient(client)
	ctx := context.TODO()
	err := client.Run(ctx, func(ctx context.Context) error {
		_, err := api.ContactsResolveUsername(ctx, &tg.ContactsResolveUsernameRequest{
			Username: "@self",
		})
		return err
	})
	if err != nil {
		panic(err)
	}
}
