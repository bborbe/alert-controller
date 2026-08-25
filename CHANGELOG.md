# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

## Unreleased

- chore: update Go to 1.27.0 and github.com/bborbe/alert to v1.8.25, github.com/bborbe/cron to v1.8.27, github.com/bborbe/errors to v1.5.21, github.com/bborbe/http to v1.26.24, github.com/bborbe/k8s to v1.14.14, github.com/bborbe/log to v1.6.25, github.com/bborbe/metrics to v0.5.15, github.com/bborbe/run to v1.9.37, github.com/bborbe/sentry to v1.9.27, github.com/bborbe/service to v1.10.9, github.com/bborbe/time to v1.27.10, k8s.io/api to v0.36.4, k8s.io/apiextensions-apiserver to v0.36.4, k8s.io/apimachinery to v0.36.4, k8s.io/client-go to v0.36.4

## v0.2.2

- chore: Bump errcheck to v1.20.0 and golangci-lint to v2.13.1 for Go 1.27 support

## v0.2.1

- update Go to 1.26.6 and update dependencies, fixing GO-2026-6179, GO-2026-6180, CVE-2026-56864, CVE-2026-56865, GO-2026-5026, GO-2026-5972, GO-2026-6089, GO-2026-6090, GO-2026-6218

## v0.2.0

- feat: Report build version as a label on the build_info metric; replace private metrics package with github.com/bborbe/metrics

## v0.1.1

- chore: Update Go dependencies to latest

## v0.1.0

- Initial release — extracted from `bborbe/quant` (`monitoring/alert-controller`) into its own repo; module path `github.com/bborbe/alert-controller`, publish-only image build (`docker.io/bborbe/alert-controller`).
