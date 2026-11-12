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

package driverconfig

// Config holds the internal driver runtime configuration.
// Use versioned APIs for any external-facing interaction.
type Config struct {
	Kubeconfig                            string
	HostnameOverride                      string
	BindAddress                           string
	ReservedCPUs                          string
	CPUDeviceMode                         string
	GroupBy                               string
	ExposePCIeRoots                       bool
	SysFSOverlay                          string
	KubeletRootDir                        string
	PublishNodeAllocatableResourceMapping bool
	Allocator                             string
}

// LogValues returns key-value pairs for structured logging of the config.
func (c Config) LogValues() []any {
	return []any{
		"kubeconfig", c.Kubeconfig,
		"bindAddress", c.BindAddress,
		"cpuDeviceMode", c.CPUDeviceMode,
		"groupBy", c.GroupBy,
		"reservedCPUs", c.ReservedCPUs,
		"hostnameOverride", c.HostnameOverride,
		"exposePCIeRoots", c.ExposePCIeRoots,
		"sysfsOverlay", c.SysFSOverlay,
		"kubeletRootDir", c.KubeletRootDir,
		"publishNodeAllocatableResourceMapping", c.PublishNodeAllocatableResourceMapping,
		"allocator", c.Allocator,
	}
}
