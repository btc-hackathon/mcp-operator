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
	"fmt"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// ConditionReason is the type of reason
type ConditionReason string

const (
	// ServiceReconciliationFailure - Unable to determine the error
	ServiceReconciliationFailure ConditionReason = "ReconciliationFailure"
	// SuccessfulDeployedReason ...
	SuccessfulDeployedReason ConditionReason = "AtLeastOnePodAvailable"
	// FailedDeployedReason ...
	FailedDeployedReason ConditionReason = "NoPodAvailable"
	// ProvisioningInProgressReason ...
	ProvisioningInProgressReason ConditionReason = "RequestedReplicasNotEqualToAvailableReplicas"
	// FailedProvisioningReason ...
	FailedProvisioningReason ConditionReason = "UnrecoverableError"
	// FinishedProvisioningReason ...
	FinishedProvisioningReason ConditionReason = "RequestedReplicasEqualToAvailableReplicas"
	// DeploymentNotAvailable ...
	DeploymentNotAvailable ConditionReason = "DeploymentNotAvailable"
	// RouteProcessed ...
	RouteProcessed ConditionReason = "RouteProcessed"
	// RouteCreationFailureReason - Unable to properly create Route
	RouteCreationFailureReason ConditionReason = "RouteCreationFailure"
	MCPServerDeployed          ConditionReason = "MCPServerDeployed"
)

// ConditionType ...
type ConditionType string

const (
	Ready ConditionType = "Ready"
)

const (
	// ReconciliationAfterThirty ...
	ReconciliationAfterThirty = time.Second * 30
	// ReconciliationAfterTen ...
	ReconciliationAfterTen = time.Second * 10
	// ReconciliationAfterFive ...
	ReconciliationAfterFive = time.Second * 5
)

// ReconciliationError ...
type ReconciliationError struct {
	reason                 ConditionReason
	reconciliationInterval time.Duration
	innerError             error
}

// String stringer implementation
func (e ReconciliationError) String() string {
	return e.innerError.Error()
}

// Error error implementation
func (e ReconciliationError) Error() string {
	return e.innerError.Error()
}

// ErrorForDeploymentNotReachable ...
func ErrorForDeploymentNotReachable(instance string) ReconciliationError {
	return ReconciliationError{
		reason:                 DeploymentNotAvailable,
		reconciliationInterval: ReconciliationAfterTen,
		innerError:             fmt.Errorf("Deployment is not yet available for MCPServer instance %s ", instance),
	}
}

// ErrorForRouteCreation ...
func ErrorForRouteCreation(err error) ReconciliationError {
	return ReconciliationError{
		reason:     RouteCreationFailureReason,
		innerError: fmt.Errorf("%w; Note the option to disable routes and provide manual route if needed", err),
	}
}

// ReconciliationErrorHandler ...
type ReconciliationErrorHandler struct {
}

// NewReconciliationErrorHandler ...
func NewReconciliationErrorHandler() *ReconciliationErrorHandler {
	return &ReconciliationErrorHandler{}
}

func (r *ReconciliationErrorHandler) IsReconciliationError(err error) bool {
	switch err.(type) {
	case ReconciliationError:
		return true
	}
	return false
}

func (r *ReconciliationErrorHandler) GetReconcileResultFor(err error) (ctrl.Result, error) {
	reconcileResult := ctrl.Result{}

	// reconciliation always happens if we return an error
	if r.IsReconciliationError(err) {
		reconcileError := err.(ReconciliationError)
		//r.Log.Info("Waiting for all resources to be created, re-scheduling.", "reason", reconcileError.reason, "requeueAfter", reconcileError.reconciliationInterval)
		reconcileResult.RequeueAfter = reconcileError.reconciliationInterval
		return reconcileResult, nil
	}
	return reconcileResult, err
}

func (r *ReconciliationErrorHandler) GetReasonForError(err error) ConditionReason {
	if err == nil {
		return ""
	}
	if r.IsReconciliationError(err) {
		reconcileError := err.(ReconciliationError)
		return reconcileError.reason
	}
	return ServiceReconciliationFailure
}
