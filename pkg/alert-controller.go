// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"

	"github.com/bborbe/errors"
	libtime "github.com/bborbe/time"
	"github.com/golang/glog"
)

type AlertController interface {
	Run(ctx context.Context) error
	Force(ctx context.Context) error
}

func NewController(
	provider AlertProvider,
	generator ConfigGenerator,
	configMapService ConfigMapService,
	prometheusReloader PrometheusReloader,
	waiterDuration libtime.WaiterDuration,
) AlertController {
	return &alertController{
		alertProvider:      provider,
		configGenerator:    generator,
		configMapService:   configMapService,
		prometheusReloader: prometheusReloader,
		waiterDuration:     waiterDuration,
	}
}

type alertController struct {
	alertProvider      AlertProvider
	configGenerator    ConfigGenerator
	configMapService   ConfigMapService
	prometheusReloader PrometheusReloader
	waiterDuration     libtime.WaiterDuration
}

func (c *alertController) Force(ctx context.Context) error {
	if err := c.run(ctx, 0); err != nil {
		return errors.Wrapf(ctx, err, "run failed")
	}
	return nil
}

func (c *alertController) Run(ctx context.Context) error {
	// kubelet sync period (1 minute by default) + TTL of ConfigMaps cache (1 minute by default)
	if err := c.run(ctx, 3*libtime.Minute); err != nil {
		return errors.Wrapf(ctx, err, "run failed")
	}
	return nil
}

func (c *alertController) run(ctx context.Context, duration libtime.Duration) error {
	glog.V(3).Infof("update configmap started")
	alerts, err := c.alertProvider.Get(ctx)
	if err != nil {
		return errors.Wrap(ctx, err, "get alerts failed")
	}
	newContent, err := c.configGenerator.GenerateConfig(ctx, alerts)
	if err != nil {
		return errors.Wrap(ctx, err, "generate config from alerts failed")
	}
	if glog.V(4) {
		glog.Infof("newContent: %s", newContent)
	}
	currentContent, err := c.configMapService.Get(ctx)
	if err != nil {
		glog.V(2).Infof("get current content failed: %v", err)
	}
	if glog.V(4) {
		glog.Infof("currentContent: %s", currentContent)
	}
	if currentContent == newContent {
		glog.V(3).Infof("content already uptodate => skip")
		return nil
	}
	if err := c.configMapService.Update(ctx, newContent); err != nil {
		return errors.Wrap(ctx, err, "update content failed")
	}
	glog.V(2).Infof("update alert configmap completed")

	if duration > 0 {
		glog.V(2).Infof("wait %v befor update configmap", duration)
		if err := c.waiterDuration.Wait(ctx, duration); err != nil {
			return errors.Wrapf(ctx, err, "wait for %v failed", duration)
		}
		glog.V(2).Infof("wait for %v completed", duration)
	}

	if err := c.prometheusReloader.Reload(ctx); err != nil {
		return errors.Wrap(ctx, err, "reload prometheus failed")
	}
	glog.V(2).Infof("reload prometheus completed")
	return nil
}
