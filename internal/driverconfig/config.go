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

import (
	"fmt"

	"sigs.k8s.io/yaml"
)

// Config holds the internal driver runtime configuration.
type Config struct {
	Kubeconfig       string
	HostnameOverride string
	BindAddress      string
	ReservedCPUs     string
	CPUDeviceMode    string
	GroupBy          string
	ExposePCIeRoots  bool
	SysFSOverlay     string
	// KubeletRootDir is the kubelet root directory. The plugin registration and
	// plugins directories are derived from it as <root>/plugins_registry and
	// <root>/plugins. Set it when the kubelet --root-dir is not the default
	// /var/lib/kubelet.
	KubeletRootDir string
	// PublishNodeAllocatableResourceMapping publishes KEP-5517 nodeAllocatableResources
	// mappings in ResourceSlice devices. Requires the DRANodeAllocatableResources
	// feature gate in the cluster. Defaults to false.
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

// dumpConfig names diagnostic fields and includes zero values. It is not a
// public configuration schema.
type dumpConfig struct {
	Kubeconfig                            string `json:"kubeconfig"`
	HostnameOverride                      string `json:"hostnameOverride"`
	BindAddress                           string `json:"bindAddress"`
	ReservedCPUs                          string `json:"reservedCPUs"`
	CPUDeviceMode                         string `json:"cpuDeviceMode"`
	GroupBy                               string `json:"groupBy"`
	ExposePCIeRoots                       bool   `json:"exposePCIeRoots"`
	SysFSOverlay                          string `json:"sysfsOverlay"`
	KubeletRootDir                        string `json:"kubeletRootDir"`
	PublishNodeAllocatableResourceMapping bool   `json:"publishNodeAllocatableResourceMapping"`
	Allocator                             string `json:"allocator"`
}

// Dump renders the Config as YAML, for logging a human-readable snapshot of
// the fully loaded internal configuration, including zero values.
func (c Config) Dump() string {
	out, err := yaml.Marshal(dumpConfig(c))
	if err != nil {
		return fmt.Sprintf("<!!! FAILED TO MARSHAL Config: %v !!!>", err)
	}
	return string(out)
}
