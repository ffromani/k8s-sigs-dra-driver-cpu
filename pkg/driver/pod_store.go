/*
Copyright 2025 The Kubernetes Authors.

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
package driver

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"
)

// CPUType defines whether the CPU allocation is for a Guaranteed or BestEffort container.
type CPUType string

const (
	// CPUTypeGuaranteed is for containers with guaranteed CPU allocation.
	CPUTypeGuaranteed CPUType = "Guaranteed"
	// CPUTypeShared is for containers with shared CPU allocation.
	CPUTypeShared CPUType = "Shared"
)

// ContainerCPUState holds the CPU allocation type and all claim assignments for a container.
type ContainerCPUState struct {
	cpuType       CPUType
	containerName string
	containerUID  types.UID
	// updated if cpuType is CPUTypeGuaranteed
	guaranteedCPUs cpuset.CPUSet
}

// NewContainerCPUState creates a new ContainerCPUState.
func NewContainerCPUState(cpuType CPUType, containerName string, containerUID types.UID, guaranteedCPUs cpuset.CPUSet) *ContainerCPUState {
	return &ContainerCPUState{
		cpuType:        cpuType,
		containerName:  containerName,
		containerUID:   containerUID,
		guaranteedCPUs: guaranteedCPUs,
	}
}

// PodCPUAssignments maps a container name to its CPU state.
type PodCPUAssignments map[string]*ContainerCPUState

// PodConfigStore maps a Pod's UID directly to its container-level CPU assignments.
type PodConfigStore struct {
	mu      sync.RWMutex
	configs map[types.UID]PodCPUAssignments
	allCPUs cpuset.CPUSet
	// fields derived by configs, whose value is cached for faster access
	// the assumption is that the fields are updated less often than they are
	// consumed, IOW we have more Get()s than Add/Remove pairs. Otherwise
	// it becomes more efficient to just iterate over `configs` as needed
	// vs keeping the fields updated.
	publicCPUs          cpuset.CPUSet
	sharedCPUContainers []types.UID
}

// NewPodConfigStore creates a new PodConfigStore.
func NewPodConfigStore(provider CPUInfoProvider) *PodConfigStore {
	cpuIDs := []int{}
	cpuInfo, err := provider.GetCPUInfos()
	if err != nil {
		klog.Fatalf("Fatal error getting CPU topology: %v", err)
	}
	for _, cpu := range cpuInfo {
		cpuIDs = append(cpuIDs, cpu.CpuID)
	}

	allCPUsSet := cpuset.New(cpuIDs...)

	return &PodConfigStore{
		configs:    make(map[types.UID]PodCPUAssignments),
		allCPUs:    allCPUsSet,
		publicCPUs: allCPUsSet.Clone(),
	}
}

// SetContainerState records or updates a container's CPU allocation using a state object.
func (s *PodConfigStore) SetContainerState(podUID types.UID, state *ContainerCPUState) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configs[podUID]; !ok {
		s.configs[podUID] = make(PodCPUAssignments)
	}

	// Use the container name from the state object as the map key.
	s.configs[podUID][state.containerName] = state

	mutated := false
	if state.cpuType == CPUTypeGuaranteed {
		klog.Infof("SetContainerState PodUID:%s Container:%s ContainerID:%s registered guaranteed container with CPUs %s", podUID, state.containerName, state.containerUID, state.guaranteedCPUs.String())
		// Recalculate public CPUs after any change that could affect guaranteed CPUs.
		s.recalculatePublicCPUs()
		mutated = true
	} else {
		klog.Infof("SetContainerState PodUID:%s Container:%s ContainerID:%s added to the shared set", podUID, state.containerName, state.containerUID)
		s.recalculateSharedCPUContainers()
	}

	return mutated
}

// GetContainerState retrieves a container's CPU state.
func (s *PodConfigStore) GetContainerState(podUID types.UID, containerName string) *ContainerCPUState {
	s.mu.RLock()
	defer s.mu.RUnlock()

	podAssignments, ok := s.configs[podUID]
	if !ok {
		return nil
	}
	return podAssignments[containerName]
}

// RemoveContainerState removes a container's state from the store.
// Returns true if the public CPU state was recomputed because the removal
func (s *PodConfigStore) RemoveContainerState(podUID types.UID, containerName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	podAssignments, ok := s.configs[podUID]
	if !ok {
		klog.Infof("RemoveContainerState PodUID:%s Container:%s WARNING: missing pod config in store", podUID, containerName)
		return false
	}

	state, ok := podAssignments[containerName]
	if !ok {
		klog.Infof("RemoveContainerState PodUID:%s Container:%s WARNING: missing container config in store", podUID, containerName)
		return false
	}

	klog.Infof("Removing container state for %s from PodUID %v", containerName, podUID)
	delete(podAssignments, containerName)

	// If this was the last container for the pod, clean up the pod entry itself.
	if len(podAssignments) == 0 {
		klog.Infof("Removing pod state from PodUID %v", podUID)
		delete(s.configs, podUID)
	}

	mutated := false
	if state.cpuType == CPUTypeGuaranteed {
		// If a guaranteed container was removed, its CPUs must be returned to the public pool.
		mutated = s.recalculatePublicCPUs()
		klog.Infof("RemoveContainerState PodUID:%s Container:%s ContainerID:%s Recalculated public CPUs after removing guaranteed container with CPUs %s", podUID, state.containerName, state.containerUID, state.guaranteedCPUs)
	} else {
		s.recalculateSharedCPUContainers()
		klog.Infof("RemoveContainerState PodUID:%s Container:%s ContainerID:%s Removed from the shared set", podUID, state.containerName, state.containerUID)
	}

	return mutated
}

// GetContainersWithSharedCPUs returns a list of container UIDs that have shared CPU allocation.
func (s *PodConfigStore) GetContainersWithSharedCPUs() []types.UID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sharedCPUContainers
}

// recalculateSharedCPUContainers computes the list of running shared container UIDs.
// needs to be called with a write lock held.
func (s *PodConfigStore) recalculateSharedCPUContainers() {
	sharedCPUContainers := []types.UID{}
	for _, podAssignments := range s.configs {
		for _, state := range podAssignments {
			if state.cpuType == CPUTypeShared {
				sharedCPUContainers = append(sharedCPUContainers, state.containerUID)
			}
		}
	}
	s.sharedCPUContainers = sharedCPUContainers
}

// GetPublicCPUs calculates and returns the list of CPUs not reserved by any Guaranteed containers.
// This is the pool available for Shared containers.
func (s *PodConfigStore) GetPublicCPUs() cpuset.CPUSet {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.publicCPUs
}

// recalculatePublicCPUs returns true if the public CPU set changed after recalculation
// needs to be called with a write lock held.
func (s *PodConfigStore) recalculatePublicCPUs() bool {
	guaranteedCPUs := cpuset.New()
	for _, podAssignments := range s.configs {
		for _, state := range podAssignments {
			if state.cpuType == CPUTypeGuaranteed {
				guaranteedCPUs = guaranteedCPUs.Union(state.guaranteedCPUs)
			}
		}
	}
	oldPublicCPUs := s.publicCPUs
	s.publicCPUs = s.allCPUs.Difference(guaranteedCPUs)
	same := s.publicCPUs.Equals(oldPublicCPUs)
	return !same
}
