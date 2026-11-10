// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"context"
	"os"
	"reflect"

	"github.com/bborbe/argument/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("version flag", func() {
	envKeys := []string{"BUILD_GIT_VERSION", "BUILD_GIT_COMMIT", "BUILD_DATE"}

	BeforeEach(func() {
		for _, key := range envKeys {
			Expect(os.Unsetenv(key)).To(Succeed())
		}
	})

	AfterEach(func() {
		for _, key := range envKeys {
			Expect(os.Unsetenv(key)).To(Succeed())
		}
	})

	Describe("hasVersionFlag", func() {
		It("matches only the exact argument", func() {
			Expect(hasVersionFlag([]string{"-version"})).To(BeTrue())
			Expect(hasVersionFlag([]string{"-listen", "x", "-version"})).To(BeTrue())
			Expect(hasVersionFlag([]string{"--version"})).To(BeFalse())
			Expect(hasVersionFlag([]string{"-version=true"})).To(BeFalse())
			Expect(hasVersionFlag(nil)).To(BeFalse())
		})
	})

	Describe("printVersion", func() {
		It("prints the struct-tag defaults when no environment is set", func() {
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(buf.String()).To(Equal("version=dev commit=none date=unknown\n"))
		})

		It("prints the values from the environment", func() {
			Expect(os.Setenv("BUILD_GIT_VERSION", "v9.9.9")).To(Succeed())
			Expect(os.Setenv("BUILD_GIT_COMMIT", "abc1234")).To(Succeed())
			Expect(os.Setenv("BUILD_DATE", "2026-01-02T03:04:05Z")).To(Succeed())
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(
				buf.String(),
			).To(Equal("version=v9.9.9 commit=abc1234 date=2026-01-02T03:04:05Z\n"))
		})

		It("renders an unparseable date raw and does not fail", func() {
			Expect(os.Setenv("BUILD_DATE", "not-a-date")).To(Succeed())
			var buf bytes.Buffer
			Expect((&application{}).printVersion(context.Background(), &buf)).To(Succeed())
			Expect(buf.String()).To(Equal("version=dev commit=none date=not-a-date\n"))
		})
	})

	Describe("buildValue", func() {
		It("reads the default from the struct tag rather than a literal", func() {
			type tagged struct {
				BuildGitVersion string `env:"TEST_BUILD_GIT_VERSION" default:"zzz"`
			}
			Expect(os.Unsetenv("TEST_BUILD_GIT_VERSION")).To(Succeed())
			value := &tagged{}
			defaults, err := argument.DefaultValues(context.Background(), value)
			Expect(err).NotTo(HaveOccurred())
			Expect(buildValue(reflect.TypeOf(value).Elem(), defaults, "BuildGitVersion")).
				To(Equal("zzz"))
		})
	})
})
