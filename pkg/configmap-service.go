// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"

	"github.com/bborbe/errors"
	"github.com/bborbe/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConfigMapService interface {
	Update(ctx context.Context, content string) error
	Get(ctx context.Context) (string, error)
}

func NewConfigMapService(
	configMapDeployer k8s.ConfigMapDeployer,
	namespace k8s.Namespace,
	configmapName k8s.Name,
	key string,
) ConfigMapService {
	return &configMapService{
		configMapDeployer: configMapDeployer,
		namespace:         namespace,
		configmapName:     configmapName,
		key:               key,
	}
}

type configMapService struct {
	configMapDeployer k8s.ConfigMapDeployer
	namespace         k8s.Namespace
	configmapName     k8s.Name
	key               string
}

func (c *configMapService) Update(ctx context.Context, newContent string) error {
	configMap, err := c.createConfigMap(ctx, newContent)
	if err != nil {
		return errors.Wrap(ctx, err, "create config map failed")
	}
	if err := c.configMapDeployer.Deploy(ctx, *configMap); err != nil {
		return errors.Wrap(ctx, err, "deploy configmap failed")
	}
	return nil
}

func (c *configMapService) createConfigMap(
	ctx context.Context,
	content string,
) (*corev1.ConfigMap, error) {
	metaBuilder := k8s.NewObjectMetaBuilder()
	metaBuilder.SetComponent("monitoring")
	metaBuilder.SetName(c.configmapName)
	metaBuilder.SetNamespace(c.namespace)
	metadata, err := metaBuilder.Build(ctx)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "build metadata failed")
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: *metadata,
		Data: map[string]string{
			c.key: content,
		},
	}, nil
}

func (c *configMapService) Get(ctx context.Context) (string, error) {
	configMap, err := c.configMapDeployer.Get(ctx, c.namespace, c.configmapName)
	if err != nil {
		return "", errors.Wrap(ctx, err, "get config map failed")
	}
	content, ok := configMap.Data[c.key]
	if !ok {
		return "", errors.Errorf(ctx, "key '%s' not found", c.key)
	}
	return content, nil
}
