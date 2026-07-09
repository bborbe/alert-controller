// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg_test

import (
	v1 "github.com/bborbe/alert/k8s/apis/monitoring.benjamin-borbe.de/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/alert-controller/pkg"
)

var _ = Describe("ConvertAlertToRuleNode", func() {
	var node pkg.RuleNode
	var alert v1.Alert
	BeforeEach(func() {
		alert = v1.Alert{
			Spec: v1.AlertSpec{
				Expression: "kafka_syncproducer_success_counter > 0",
			},
		}
	})
	JustBeforeEach(func() {
		node = pkg.ConvertAlertToRuleNode(alert)
	})
	Context("Expression", func() {
		It("returns no error", func() {
			Expect(node.Expr).To(Equal("kafka_syncproducer_success_counter > 0"))
		})
	})
	Context("Expression with comment", func() {
		BeforeEach(func() {
			alert.Spec.Expression = "kafka_syncproducer_success_counter > 0 # banana"
		})
		It("returns no error", func() {
			Expect(node.Expr).To(Equal("kafka_syncproducer_success_counter > 0"))
		})
	})
	Context("Expression multi spaces", func() {
		BeforeEach(func() {
			alert.Spec.Expression = "kafka_syncproducer_success_counter     >     0"
		})
		It("returns no error", func() {
			Expect(node.Expr).To(Equal("kafka_syncproducer_success_counter > 0"))
		})
	})
})
