// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"fmt"
	"net/http"

	"github.com/bborbe/errors"
	"github.com/golang/glog"
)

type PrometheusReloader interface {
	Reload(ctx context.Context) error
}

func NewPrometheusReloader(
	httpClient *http.Client,
	prometheusURL string,
	prometheusUsername string,
	prometheusPassword string,
) PrometheusReloader {
	return &prometheusReloader{
		httpClient:         httpClient,
		prometheusURL:      prometheusURL,
		prometheusUsername: prometheusUsername,
		prometheusPassword: prometheusPassword,
	}
}

type prometheusReloader struct {
	httpClient         *http.Client
	prometheusURL      string
	prometheusUsername string
	prometheusPassword string
}

func (p *prometheusReloader) Reload(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		fmt.Sprintf("%s/-/reload", p.prometheusURL),
		nil,
	)
	if err != nil {
		return errors.Wrap(ctx, err, "create request failed")
	}
	if p.prometheusUsername != "" && p.prometheusPassword != "" {
		req.SetBasicAuth(p.prometheusUsername, p.prometheusPassword)
	}
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(ctx, err, "do request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return errors.Errorf(ctx, "statusCode != 2xx")
	}
	glog.V(2).Infof("trigger prometheus reload completed")
	return nil
}
