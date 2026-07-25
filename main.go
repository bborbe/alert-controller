// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/bborbe/alert"
	libcron "github.com/bborbe/cron"
	"github.com/bborbe/errors"
	libhttp "github.com/bborbe/http"
	"github.com/bborbe/k8s"
	libmetrics "github.com/bborbe/metrics"
	"github.com/bborbe/run"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	libtime "github.com/bborbe/time"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	k8s_kubernetes "k8s.io/client-go/kubernetes"

	"github.com/bborbe/alert-controller/pkg"
	"github.com/bborbe/alert-controller/pkg/factory"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN          string            `required:"true"  arg:"sentry-dsn"               env:"SENTRY_DSN"               usage:"SentryDSN"                                                 display:"length"`
	SentryProxy        string            `required:"false" arg:"sentry-proxy"             env:"SENTRY_PROXY"             usage:"Sentry Proxy"`
	Listen             string            `required:"true"  arg:"listen"                   env:"LISTEN"                   usage:"address to listen to"`
	Kubeconfig         string            `required:"false" arg:"kubeconfig"               env:"KUBECONFIG"               usage:"Path to k8s config"`
	CronExpression     string            `required:"false" arg:"cron-schedule-expression" env:"CRON_SCHEDULE_EXPRESSION" usage:"Cron schedule expression to determine when service is run"                  default:"@every 1m"`
	PrometheusURL      string            `required:"true"  arg:"prometheus-url"           env:"PROMETHEUS_URL"           usage:"Prometheus url"`
	PrometheusUsername string            `required:"false" arg:"prometheus-user"          env:"PROMETHEUS_USER"          usage:"Prometheus user"`
	PrometheusPassword string            `required:"false" arg:"prometheus-password"      env:"PROMETHEUS_PASSWORD"      usage:"Prometheus password"                                       display:"length"`
	BuildGitVersion    string            `required:"false" arg:"build-git-version"        env:"BUILD_GIT_VERSION"        usage:"Build Git version (git describe --tags --always --dirty)"                   default:"dev"`
	BuildGitCommit     string            `required:"false" arg:"build-git-commit"         env:"BUILD_GIT_COMMIT"         usage:"Build Git commit hash"                                                      default:"none"`
	BuildDate          *libtime.DateTime `required:"false" arg:"build-date"               env:"BUILD_DATE"               usage:"Build timestamp (RFC3339)"`
}

func (a *application) Run(ctx context.Context, sentryClient libsentry.Client) error {
	libmetrics.NewBuildInfoMetrics().SetBuildInfo(a.BuildGitVersion, a.BuildGitCommit, a.BuildDate)

	k8sClientset, err := k8s.CreateClientset(a.Kubeconfig)
	if err != nil {
		return errors.Wrap(ctx, err, "build k8s clientset failed")
	}

	eventHandlerAlert := alert.NewAlertEventHandler()

	httpClientBuilder := libhttp.NewClientBuilder()
	httpClientBuilder.WithTimeout(30 * time.Second)
	httpClientBuilder.WithRetry(3, time.Second)
	httpClient, err := httpClientBuilder.Build(ctx)
	if err != nil {
		return errors.Wrapf(ctx, err, "create httpClient failed")
	}

	trigger := run.NewTrigger()

	return service.Run(
		ctx,
		a.setupResourceDefinition(trigger),
		run.Triggered(a.watchAlerts(eventHandlerAlert), trigger.Done()),
		run.Triggered(
			a.createCron(httpClient, sentryClient, eventHandlerAlert, k8sClientset),
			trigger.Done(),
		),
		a.createHTTPServer(httpClient, eventHandlerAlert, k8sClientset),
	)
}

func (a *application) setupResourceDefinition(trigger run.Fire) run.Func {
	return factory.CreateSetupResourceDefinition(a.Kubeconfig, trigger)
}

func (a *application) watchAlerts(eventHandlerAlert alert.AlertEventHandler) run.Func {
	return factory.CreateAlertWatcher(a.Kubeconfig, eventHandlerAlert)
}

func (a *application) createCron(
	httpClient *http.Client,
	sentryClient libsentry.Client,
	eventHandlerAlert alert.AlertEventHandler,
	k8sClientset k8s.Interface,
) run.Func {
	return factory.CreateCron(
		httpClient,
		sentryClient,
		eventHandlerAlert,
		k8sClientset,
		a.PrometheusURL,
		a.PrometheusUsername,
		a.PrometheusPassword,
		libcron.Expression(a.CronExpression),
	)
}

func (a *application) createHTTPServer(
	httpClient *http.Client,
	eventHandlerAlert alert.AlertEventHandler,
	k8sClientset k8s_kubernetes.Interface,
) run.Func {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		prometheusReloader := pkg.NewPrometheusReloader(
			libhttp.CreateDefaultHttpClient(),
			a.PrometheusURL,
			a.PrometheusUsername,
			a.PrometheusPassword,
		)

		router := mux.NewRouter()
		router.Path("/healthz").Handler(libhttp.NewPrintHandler("OK"))
		router.Path("/readiness").Handler(libhttp.NewPrintHandler("OK"))
		router.Path("/metrics").Handler(promhttp.Handler())
		router.Path("/setloglevel/{level}").Handler(factory.CreateSetLoglevelHandler(ctx))
		router.Path("/reload").
			Handler(libhttp.NewBackgroundRunHandler(ctx, prometheusReloader.Reload))
		router.Path("/trigger/force").
			Handler(libhttp.NewBackgroundRunHandler(ctx, factory.CreateController(httpClient, eventHandlerAlert, k8sClientset, a.PrometheusURL, a.PrometheusUsername, a.PrometheusPassword).Force))
		router.Path("/trigger").
			Handler(libhttp.NewBackgroundRunHandler(ctx, factory.CreateController(httpClient, eventHandlerAlert, k8sClientset, a.PrometheusURL, a.PrometheusUsername, a.PrometheusPassword).Run))
		router.Path("/list").Handler(libhttp.NewErrorHandler(
			libhttp.NewJsonHandler(
				libhttp.JsonHandlerFunc(
					func(ctx context.Context, req *http.Request) (interface{}, error) {
						alerts, err := eventHandlerAlert.Get(ctx)
						if err != nil {
							return nil, errors.Wrap(ctx, err, "get alert failed")
						}
						return alerts, nil
					},
				),
			),
		))
		router.Path("/rules").Handler(factory.CreateRulesHandler(eventHandlerAlert))

		glog.V(2).Infof("starting http server listen on %s", a.Listen)
		return libhttp.NewServer(
			a.Listen,
			router,
		).Run(ctx)
	}
}
