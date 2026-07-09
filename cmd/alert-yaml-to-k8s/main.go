// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	v1 "github.com/bborbe/alert/k8s/apis/monitoring.benjamin-borbe.de/v1"
	"github.com/bborbe/errors"
	"github.com/bborbe/k8s"
	"github.com/bborbe/sentry"
	"github.com/bborbe/service"
	"github.com/golang/glog"
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/bborbe/alert-controller/pkg"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN   string `required:"true"  arg:"sentry-dsn"   env:"SENTRY_DSN"   usage:"SentryDSN"        display:"length"`
	SentryProxy string `required:"false" arg:"sentry-proxy" env:"SENTRY_PROXY" usage:"Sentry Proxy"`
	File        string `required:"true"  arg:"file"         env:"FILE"         usage:"alert rules yaml"`
}

func (a *application) Run(
	ctx context.Context,
	sentryClient sentry.Client,
) error {

	content, err := os.ReadFile(a.File)
	if err != nil {
		return errors.Wrapf(ctx, err, "read %s failed", a.File)
	}

	var ruleGroups pkg.RuleGroups

	if err := yaml.Unmarshal(content, &ruleGroups); err != nil {
		return err
	}

	names := map[string]int{}

	for _, g := range ruleGroups.Groups {
		for _, r := range g.Rules {
			if r.Alert == "" {
				continue
			}
			name := generateName(r.Alert)
			counter := names[name]
			counter++
			names[name] = counter

			alert, err := createAlert(ctx, k8s.Name(fmt.Sprintf("%s%d", name, counter)), r)
			if err != nil {
				return errors.Wrap(ctx, err, "create alert failed")
			}

			content, err := yaml.Marshal(alert)
			if err != nil {
				return errors.Wrap(ctx, err, "marshal alert failed")
			}

			filename := fmt.Sprintf("%s%d.yaml", name, counter)
			if err := os.WriteFile(filename, content, 0600); err != nil {
				return err
			}
			glog.V(2).Infof("write %s completed", filename)
		}
	}
	glog.V(2).Infof("generate completed")
	return nil
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")

var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func generateName(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}-${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}-${2}")
	return strings.ToLower(snake)
}

func createAlert(ctx context.Context, name k8s.Name, ruleNode pkg.RuleNode) (*v1.Alert, error) {
	metaBuilder := k8s.NewObjectMetaBuilder()
	metaBuilder.SetComponent("monitoring")
	metaBuilder.SetName(name)
	metaBuilder.SetNamespace("monitoring")
	metadata, err := metaBuilder.Build(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "build metadata failed")
	}
	return &v1.Alert{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Alert",
			APIVersion: "monitoring.benjamin-borbe.de/v1",
		},
		ObjectMeta: *metadata,
		Spec: v1.AlertSpec{
			Name:        ruleNode.Alert,
			Annotations: ruleNode.Annotations,
			Expression:  ruleNode.Expr,
			For:         ruleNode.For,
			Labels:      ruleNode.Labels,
		},
	}, nil
}
