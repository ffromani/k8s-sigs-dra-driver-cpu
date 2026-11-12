/*
Copyright The Kubernetes Authors.

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

package v1alpha1

import (
	"fmt"

	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

const (
	APIVersion = "v1alpha1"
)

// CPUDeviceMode selects how CPUs are exposed as DRA devices.
type CPUDeviceMode string

const (
	CPUDeviceModeGrouped    CPUDeviceMode = "grouped"
	CPUDeviceModeIndividual CPUDeviceMode = "individual"
)

// GroupBy selects the topology boundary for grouped devices.
type GroupBy string

const (
	GroupByNUMANode GroupBy = "numanode"
	GroupBySocket   GroupBy = "socket"
	GroupByMachine  GroupBy = "machine"
)

// Allocator selects how physical CPUs are assigned to claims.
type Allocator string

const (
	AllocatorCPUManager Allocator = "cpumanager"
	AllocatorExternal   Allocator = "external"
)

// Config holds the driver runtime configuration.
type Config struct {
	Kubeconfig       *string        `json:"kubeconfig,omitempty"`
	HostnameOverride *string        `json:"hostnameOverride,omitempty"`
	ReservedCPUs     *string        `json:"reservedCPUs,omitempty"`
	CPUDeviceMode    *CPUDeviceMode `json:"cpuDeviceMode,omitempty"`
	GroupBy          *GroupBy       `json:"groupBy,omitempty"`
	SysFSOverlay     *string        `json:"sysfsOverlay,omitempty"`
	// PublishNodeAllocatableResourceMapping publishes KEP-5517 nodeAllocatableResources
	// mappings in ResourceSlice devices. Requires the DRANodeAllocatableResources
	// feature gate in the cluster. Defaults to false.
	PublishNodeAllocatableResourceMapping *bool      `json:"publishNodeAllocatableResourceMapping,omitempty"`
	Allocator                             *Allocator `json:"allocator,omitempty"`
}

// dumpConfig mirrors Config field-for-field but drops the omitempty json
// tags, so Dump also prints zero values.
type dumpConfig struct {
	Kubeconfig                            string `json:"kubeconfig"`
	HostnameOverride                      string `json:"hostnameOverride"`
	ReservedCPUs                          string `json:"reservedCPUs"`
	CPUDeviceMode                         string `json:"cpuDeviceMode"`
	GroupBy                               string `json:"groupBy"`
	SysFSOverlay                          string `json:"sysfsOverlay"`
	PublishNodeAllocatableResourceMapping bool   `json:"publishNodeAllocatableResourceMapping"`
	Allocator                             string `json:"allocator"`
}

// Dump renders the Config as YAML, for logging a human-readable snapshot of
// the fully loaded configuration. Zero values are included, unlike
// marshalling Config directly, since they reflect real runtime state.
func (c Config) Dump() string {
	out, err := yaml.Marshal(dumpConfig{
		Kubeconfig:                            ptr.Deref(c.Kubeconfig, ""),
		HostnameOverride:                      ptr.Deref(c.HostnameOverride, ""),
		ReservedCPUs:                          ptr.Deref(c.ReservedCPUs, ""),
		CPUDeviceMode:                         string(ptr.Deref(c.CPUDeviceMode, "")),
		GroupBy:                               string(ptr.Deref(c.GroupBy, "")),
		SysFSOverlay:                          ptr.Deref(c.SysFSOverlay, ""),
		PublishNodeAllocatableResourceMapping: ptr.Deref(c.PublishNodeAllocatableResourceMapping, false),
		Allocator:                             string(ptr.Deref(c.Allocator, "")),
	})
	if err != nil {
		return fmt.Sprintf("<!!! FAILED TO MARSHAL Config: %v !!!>", err)
	}
	return string(out)
}
