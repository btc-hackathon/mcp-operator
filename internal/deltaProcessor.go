/*

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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeltaProcessor would compare the request to desired resource and return the delta that needs to be process.
type DeltaProcessor struct {
}

func NewDeltaProcessor() *DeltaProcessor {
	return &DeltaProcessor{}
}

func (d *DeltaProcessor) ComputeDelta(comparator ResourceComparator, desiredResource client.Object, existingResource client.Object) ResourceDelta {
	var added bool
	var updated bool
	var removed bool

	if !(IsNil(desiredResource) && IsNil(existingResource)) {
		if IsNotNil(desiredResource) && IsNil(existingResource) {
			added = true
		} else if IsNil(desiredResource) && IsNotNil(existingResource) {
			removed = true
		} else if !comparator(existingResource, desiredResource) {
			updated = true
		}
	}

	return &resourceDelta{
		Added:   added,
		Updated: updated,
		Removed: removed,
	}
}

type ResourceDelta interface {
	HasChanges() bool
	IsAdded() bool
	IsUpdated() bool
	IsRemoved() bool
}

type resourceDelta struct {
	Added   bool
	Updated bool
	Removed bool
}

func (delta *resourceDelta) HasChanges() bool {
	return delta.Added || delta.Updated || delta.Removed
}

func (delta *resourceDelta) IsAdded() bool {
	return delta.Added
}

func (delta *resourceDelta) IsUpdated() bool {
	return delta.Updated
}

func (delta *resourceDelta) IsRemoved() bool {
	return delta.Removed
}
