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
	mcpv1alpha1 "github.com/opendatahub-io/mcp-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RawKubeReconciler interface {
	Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error
}

type rawKubeReconciler struct {
	client               client.Client
	deploymentReconciler DeploymentReconciler
	serviceReconciler    ServiceReconciler
}

func NewRawKubeReconciler(client client.Client) RawKubeReconciler {
	return &rawKubeReconciler{
		client:               client,
		deploymentReconciler: NewDeploymentReconciler(client),
		serviceReconciler:    NewServiceReconciler(client),
	}
}

func (r *rawKubeReconciler) Reconcile(ctx context.Context, logger logr.Logger, mspServer *mcpv1alpha1.MCPServer, mcpServerTemplate *mcpv1alpha1.MCPServerTemplate) error {

	err := r.deploymentReconciler.Reconcile(ctx, logger, mspServer, mcpServerTemplate)
	if err != nil {
		return err
	}

	err = r.serviceReconciler.Reconcile(ctx, logger, mspServer, mcpServerTemplate)
	if err != nil {
		return err
	}
	return nil
}
