package youtube

import (
	"time"

	"resty.dev/v3"
)

func newClient() *resty.Client {
	client := resty.New().
		SetTimeout(15 * time.Second).
		SetRetryCount(2).
		SetRetryWaitTime(500 * time.Millisecond)

	return client
}
