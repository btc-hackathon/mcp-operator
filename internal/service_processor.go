/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"context"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ServiceProcessor struct {
	client.Client
}

func NewServiceProcessor(client client.Client) *ServiceProcessor {
	return &ServiceProcessor{
		client,
	}
}

func (d *ServiceProcessor) FetchService(ctx context.Context, logger logr.Logger, key types.NamespacedName) (*v1.Service, error) {
	service := &v1.Service{}
	err := d.Client.Get(ctx, key, service)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Service not found")
		return nil, nil
	} else if err != nil {
		logger.Error(err, "Unable to fetch the Service")
		return nil, err
	}
	logger.Info("Successfully fetch Service")
	return service, nil
}
