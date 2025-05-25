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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type DeploymentProcessor struct {
	client.Client
}

func NewDeploymentProcessor(client client.Client) *DeploymentProcessor {
	return &DeploymentProcessor{
		client,
	}
}

func (d *DeploymentProcessor) IsDeploymentAvailable(ctx context.Context, logger logr.Logger, key types.NamespacedName) (bool, error) {
	deployment, err := d.FetchDeployment(ctx, logger, key)
	if err != nil {
		return false, err
	} else if deployment == nil {
		return false, nil
	}
	return deployment.Status.AvailableReplicas > 0, nil
}

func (d *DeploymentProcessor) FetchDeployment(ctx context.Context, logger logr.Logger, key types.NamespacedName) (*v1.Deployment, error) {
	deployment := &v1.Deployment{}
	err := d.Client.Get(ctx, key, deployment)
	if err != nil && errors.IsNotFound(err) {
		logger.Info("Deployment not found")
		return nil, nil
	} else if err != nil {
		logger.Error(err, "Unable to fetch the Deployment")
		return nil, err
	}
	logger.Info("Successfully fetch Deployment")
	return deployment, nil
}
