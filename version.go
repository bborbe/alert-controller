// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"

	"github.com/bborbe/argument/v2"
	"github.com/bborbe/errors"
)

const versionFlag = "-version"

// hasVersionFlag reports whether args contains the exact -version argument.
func hasVersionFlag(args []string) bool {
	return slices.Contains(args, versionFlag)
}

// printVersion writes the build version, commit and date to w. The values come
// from the environment variables and struct-tag defaults declared on
// application; an unset date renders as "unknown".
func (a *application) printVersion(ctx context.Context, w io.Writer) error {
	defaults, err := argument.DefaultValues(ctx, a)
	if err != nil {
		return errors.Wrap(ctx, err, "read argument defaults failed")
	}
	typ := reflect.TypeOf(a).Elem()
	version := buildValue(typ, defaults, "BuildGitVersion")
	commit := buildValue(typ, defaults, "BuildGitCommit")
	date := buildValue(typ, defaults, "BuildDate")
	if date == "" {
		date = "unknown"
	}
	_, _ = fmt.Fprintf(w, "version=%s commit=%s date=%s\n", version, commit, date)
	return nil
}

// buildValue returns the environment value for field when set, otherwise the
// field's struct-tag default.
func buildValue(typ reflect.Type, defaults map[string]interface{}, field string) string {
	f, ok := typ.FieldByName(field)
	if !ok {
		return ""
	}
	if value := os.Getenv(f.Tag.Get("env")); value != "" {
		return value
	}
	if value, ok := defaults[field].(string); ok {
		return value
	}
	return ""
}
