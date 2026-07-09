// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"regexp"
	"strings"

	v1 "github.com/bborbe/alert/k8s/apis/monitoring.benjamin-borbe.de/v1"
)

func ConvertAlertToRuleNode(alert v1.Alert) RuleNode {
	return RuleNode{
		Alert:       alert.Spec.Name,
		Expr:        cleanUpExpression(alert.Spec.Expression),
		For:         alert.Spec.For,
		Labels:      alert.Spec.Labels,
		Annotations: alert.Spec.Annotations,
	}
}

var removeCommentRegex = regexp.MustCompile(`\s*#.*$`)

var removeMultispace = regexp.MustCompile(`\s+`)

func cleanUpExpression(expression string) string {
	expression = removeCommentRegex.ReplaceAllString(expression, "")
	expression = strings.ReplaceAll(expression, "\n", " ")
	expression = strings.ReplaceAll(expression, "\t", " ")
	expression = removeMultispace.ReplaceAllString(expression, " ")
	return expression
}
