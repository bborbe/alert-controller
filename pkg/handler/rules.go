// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package handler

import (
	"context"
	"fmt"
	"net/http"

	v1 "github.com/bborbe/alert/k8s/apis/monitoring.benjamin-borbe.de/v1"
	"github.com/bborbe/errors"
	libhttp "github.com/bborbe/http"
)

type ConfigGenerator interface {
	GenerateConfig(ctx context.Context, alerts v1.Alerts) (string, error)
}

type AlertEventHandler interface {
	Get(ctx context.Context) ([]v1.Alert, error)
}

func NewRulesHandler(
	alertEventHandler AlertEventHandler,
	configGenerator ConfigGenerator,
) libhttp.WithError {
	return libhttp.WithErrorFunc(
		func(ctx context.Context, resp http.ResponseWriter, req *http.Request) error {
			alertsList, err := alertEventHandler.Get(ctx)
			if err != nil {
				return errors.Wrap(ctx, err, "get alert failed")
			}
			alerts := v1.Alerts(alertsList)
			output, err := configGenerator.GenerateConfig(ctx, alerts)
			if err != nil {
				return errors.Wrapf(ctx, err, "generate failed")
			}
			_, _ = fmt.Fprintln(resp, output)
			return nil
		},
	)
}
