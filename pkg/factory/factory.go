// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package factory

import (
	"context"
	"net/http"

	"github.com/bborbe/alert"
	libcron "github.com/bborbe/cron"
	"github.com/bborbe/errors"
	libhttp "github.com/bborbe/http"
	"github.com/bborbe/k8s"
	"github.com/bborbe/run"
	"github.com/bborbe/sentry"
	libtime "github.com/bborbe/time"
	"github.com/golang/glog"
	k8s_kubernetes "k8s.io/client-go/kubernetes"

	"github.com/bborbe/alert-controller/pkg"
	"github.com/bborbe/alert-controller/pkg/cron"
	"github.com/bborbe/alert-controller/pkg/handler"
)

func CreateCron(
	httpClient *http.Client,
	sentryClient sentry.Client,
	eventHandlerAlert alert.AlertEventHandler,
	k8sClientset k8s_kubernetes.Interface,
	prometheusURL string,
	prometheusUsername string,
	prometheusPassword string,
	cronExpression libcron.Expression,
) run.Func {
	return func(ctx context.Context) error {
		glog.V(2).Infof("cron started")
		return cron.NewExpressionCron(
			sentryClient,
			CreateController(
				httpClient,
				eventHandlerAlert,
				k8sClientset,
				prometheusURL,
				prometheusUsername,
				prometheusPassword,
			).Run,
			cronExpression,
		).Run(ctx)
	}
}

func CreateSetupResourceDefinition(
	kubeConfig string,
	trigger run.Fire,
) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		k8sConnector := pkg.NewK8sConnector(kubeConfig)
		if err := k8sConnector.SetupCustomResourceDefinition(ctx); err != nil {
			return errors.Wrap(ctx, err, "setup resource definition failed")
		}
		trigger.Fire()
		<-ctx.Done()
		return nil
	}
}

func CreateAlertWatcher(
	kubeConfig string,
	eventHandlerAlert alert.AlertEventHandler,
) run.Func {
	return func(ctx context.Context) error {
		k8sConnector := pkg.NewK8sConnector(kubeConfig)
		return k8sConnector.Listen(
			ctx,
			alert.NewAlertResourceEventHandler(
				ctx,
				eventHandlerAlert,
			),
		)
	}
}

func CreateController(
	httpClient *http.Client,
	eventHandlerAlert alert.AlertEventHandler,
	k8sClientset k8s_kubernetes.Interface,
	prometheusURL string,
	prometheusUsername string,
	prometheusPassword string,
) pkg.AlertController {
	return pkg.NewController(
		eventHandlerAlert,
		pkg.NewConfigGenerator(),
		pkg.NewConfigMapService(
			k8s.NewConfigMapDeployer(k8sClientset),
			"monitoring",
			"alerts",
			"rules.yaml",
		),
		pkg.NewPrometheusReloader(
			httpClient,
			prometheusURL,
			prometheusUsername,
			prometheusPassword,
		),
		libtime.NewWaiterDuration(),
	)
}

func CreateRulesHandler(alertEventHandler alert.AlertEventHandler) http.Handler {
	configGenerator := pkg.NewConfigGenerator()
	return libhttp.NewErrorHandler(handler.NewRulesHandler(alertEventHandler, configGenerator))
}
