// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"time"

	"github.com/bborbe/alert"
	"github.com/bborbe/alert/k8s/client/informers/externalversions"
	"github.com/bborbe/errors"
	"github.com/bborbe/k8s"
	"github.com/golang/glog"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultResync = 5 * time.Minute
	name          = "alerts.monitoring.benjamin-borbe.de"
)

//counterfeiter:generate -o ../mocks/k8s-connector.go --fake-name K8sConnector . K8sConnector
type K8sConnector interface {
	SetupCustomResourceDefinition(ctx context.Context) error
	Listen(
		ctx context.Context,
		resourceEventHandler cache.ResourceEventHandler,
	) error
}

func NewK8sConnector(
	kubeconfig string,
) K8sConnector {
	return &k8sConnector{
		kubeconfig: kubeconfig,
	}
}

type k8sConnector struct {
	kubeconfig string
}

func (k *k8sConnector) Listen(
	ctx context.Context,
	resourceEventHandler cache.ResourceEventHandler,
) error {
	clientset, err := alert.CreateClientset(ctx, k.kubeconfig)
	if err != nil {
		return errors.Wrap(ctx, err, "build clientset failed")
	}
	informerFactory := externalversions.NewSharedInformerFactory(clientset, defaultResync)
	_, err = informerFactory.
		Monitoring().
		V1().
		Alerts().
		Informer().
		AddEventHandler(resourceEventHandler)
	if err != nil {
		return errors.Wrap(ctx, err, "add event handler failed")
	}

	stopCh := make(chan struct{})
	glog.V(2).Infof("listen for events")
	informerFactory.Start(stopCh)
	select {
	case <-ctx.Done():
		glog.V(0).Infof("listen canceled")
	case <-stopCh:
		glog.V(0).Infof("listen stopped")
	}
	return nil
}

func (k *k8sConnector) SetupCustomResourceDefinition(ctx context.Context) error {
	config, err := k8s.CreateConfig(k.kubeconfig)
	if err != nil {
		return errors.Wrap(ctx, err, "build k8s config failed")
	}
	clientset, err := apiextensionsClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(ctx, err, "build clientset failed")
	}
	customResourceDefinition, err := clientset.ApiextensionsV1().
		CustomResourceDefinitions().
		Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		glog.V(2).Infof("CustomResourceDefinition '%s' not found (%v) => create", name, err)
		if err := k.createCrd(ctx, clientset); err != nil {
			return errors.Wrap(ctx, err, "create crd failed")
		}
		return nil
	}
	if err := k.updateCrd(ctx, customResourceDefinition, clientset); err != nil {
		return errors.Wrap(ctx, err, "create crd failed")
	}
	return nil
}

func (k *k8sConnector) updateCrd(
	ctx context.Context,
	customResourceDefinition *v1.CustomResourceDefinition,
	clientset *apiextensionsClient.Clientset,
) error {
	customResourceDefinition.Spec = createSpec()
	if _, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, customResourceDefinition, metav1.UpdateOptions{}); err != nil {
		return errors.Wrap(ctx, err, "update CustomResourceDefinition failed")
	}
	glog.V(2).Infof("CustomResourceDefinitions '%s' updated", name)
	return nil
}

func (k *k8sConnector) createCrd(
	ctx context.Context,
	clientset *apiextensionsClient.Clientset,
) error {
	_, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Create(
		ctx,
		&v1.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiextensions.k8s.io/v1",
				Kind:       "CustomResourceDefinition",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: createSpec(),
		},
		metav1.CreateOptions{},
	)
	if err != nil {
		return errors.Wrap(ctx, err, "create CustomResourceDefinition failed")
	}
	glog.V(2).Infof("CustomResourceDefinition '%s' created", name)
	return nil
}

func boolPointer(value bool) *bool {
	return &value
}

func createSpec() v1.CustomResourceDefinitionSpec {
	return v1.CustomResourceDefinitionSpec{
		Group: "monitoring.benjamin-borbe.de",
		Names: v1.CustomResourceDefinitionNames{
			Kind:     "Alert",
			ListKind: "AlertList",
			Plural:   "alerts",
			Singular: "alert",
		},
		Scope: "Namespaced",
		Versions: []v1.CustomResourceDefinitionVersion{
			{
				Name:    "v1",
				Served:  true,
				Storage: true,
				Schema: &v1.CustomResourceValidation{
					OpenAPIV3Schema: &v1.JSONSchemaProps{
						XPreserveUnknownFields: boolPointer(true),
					},
				},
			},
		},
	}
}
