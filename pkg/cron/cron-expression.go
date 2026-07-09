// Copyright (c) 2023 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cron

import (
	"context"

	"github.com/bborbe/cron"
	"github.com/bborbe/run"
	libsentry "github.com/bborbe/sentry"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// NewExpressionCron wraps `bborbe/cron.NewExpressionCron` with a per-tick
// sentry exception capture so transient errors don't silently disappear into
// the cron loop. Inlined from the former trading/lib/cron package; the Tx
// variant is not ported because alert-controller does not use it.
func NewExpressionCron(
	sentryClient libsentry.Client,
	action run.Func,
	expression cron.Expression,
) run.Func {
	return func(ctx context.Context) (err error) {
		return cron.NewExpressionCron(
			expression,
			run.Func(func(ctx context.Context) error {
				glog.V(3).Infof("cron started")
				if err := action(ctx); err != nil {
					sentryClient.CaptureException(
						err,
						&sentry.EventHint{
							Context:           ctx,
							OriginalException: err,
						},
						sentry.NewScope(),
					)
				}
				glog.V(3).Infof("cron completed")
				return nil
			}),
		).Run(ctx)
	}
}
