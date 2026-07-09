// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"sort"

	v1 "github.com/bborbe/alert/k8s/apis/monitoring.benjamin-borbe.de/v1"
	"github.com/bborbe/errors"
	"gopkg.in/yaml.v3"
)

type ConfigGenerator interface {
	GenerateConfig(ctx context.Context, alerts v1.Alerts) (string, error)
}

func NewConfigGenerator() ConfigGenerator {
	return &configGenerator{}
}

type configGenerator struct {
}

type RuleGroups struct {
	Groups []RuleGroup `yaml:"groups"`
}

type RuleGroup struct {
	Name  string     `yaml:"name"`
	Rules []RuleNode `yaml:"rules"`
}

type RuleNode struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

func (g *configGenerator) GenerateConfig(ctx context.Context, alerts v1.Alerts) (string, error) {
	sort.Sort(v1.AlertsSorted(alerts))

	silenceRules := []RuleNode{
		{
			Record: "always",
			Expr:   "vector(1)",
		},
		{
			Record: "never",
			Expr:   "vector(0)",
		},
		{
			Record: "is_weekend",
			Expr:   "vector(1) and (day_of_week() == 0 or day_of_week() == 6) or vector(0)", // sunday: 0, saturday: 6
		},
		{
			Record: "is_weekday",
			Expr:   "vector(1) and day_of_week() != 0 and day_of_week() != 6 or vector(0)", // sunday: 0, saturday: 6
		},
		{
			Record: "is_night",
			Expr:   "vector(1) and (hour() < 4 or hour() > 22) or vector(0)",
		},
		{
			Record: "is_day",
			Expr:   "vector(1) and hour() >= 4 and hour() <= 22 or vector(0)",
		},
		{
			Record: "office_hours",
			Expr:   "vector(1) and day_of_week() != 0 and day_of_week() != 6 and hour() > 7 and hour() < 17 or vector(0)",
		},
		{
			Alert: "QuietHours",
			Expr:  "always == 0",
			For:   "1m",
			Labels: map[string]string{
				"quiethours": "true",
			},
			Annotations: map[string]string{
				"description": "This alert fires during quiet hours. It should be blackholed by Alertmanager.",
			},
		},
	}

	alertRules := make([]RuleNode, 0, len(alerts))
	for _, alert := range alerts {
		alertRules = append(alertRules, ConvertAlertToRuleNode(alert))
	}

	groups := RuleGroups{
		Groups: []RuleGroup{
			{
				Name:  "silence-rules",
				Rules: silenceRules,
			},
			{
				Name:  "alert-rules",
				Rules: alertRules,
			},
		},
	}

	bytes, err := yaml.Marshal(groups)
	if err != nil {
		return "", errors.Wrap(ctx, err, "generate yaml failed")
	}
	return string(bytes), nil
}
