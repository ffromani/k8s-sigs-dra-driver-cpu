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

import "encoding/json"

// APIVersion is the version validated in config files.
const APIVersion = "v1alpha1"

// Config defines the v1alpha1 driver configuration file format.
type Config struct {
	APIVersion                            string  `json:"apiVersion,omitempty"` // legacy support, will be made mandatory in next versions
	Kubeconfig                            *string `json:"kubeconfig,omitempty"`
	HostnameOverride                      *string `json:"hostnameOverride,omitempty"`
	ReservedCPUs                          *string `json:"reservedCPUs,omitempty"`
	CPUDeviceMode                         *string `json:"cpuDeviceMode,omitempty"`
	GroupBy                               *string `json:"groupBy,omitempty"`
	SysFSOverlay                          *string `json:"sysfsOverlay,omitempty"`
	PublishNodeAllocatableResourceMapping *bool   `json:"publishNodeAllocatableResourceMapping,omitempty"`
	Allocator                             *string `json:"allocator,omitempty"`
}

func (cfg Config) String() string {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "<ERROR>"
	}
	return string(data)
}
