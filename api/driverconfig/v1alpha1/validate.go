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
	"path/filepath"
)

// Validate checks supplied fields without requiring omitted overlay values.
func (c Config) Validate() error {
	if c.CPUDeviceMode != nil && *c.CPUDeviceMode != CPUDeviceModeGrouped && *c.CPUDeviceMode != CPUDeviceModeIndividual {
		return fmt.Errorf("invalid cpuDeviceMode %q: must be %q or %q",
			*c.CPUDeviceMode, CPUDeviceModeGrouped, CPUDeviceModeIndividual)
	}
	if c.GroupBy != nil && *c.GroupBy != GroupByNUMANode && *c.GroupBy != GroupBySocket && *c.GroupBy != GroupByMachine {
		return fmt.Errorf("invalid groupBy %q: must be %q, %q, or %q",
			*c.GroupBy, GroupByNUMANode, GroupBySocket, GroupByMachine)
	}
	if c.Allocator != nil && *c.Allocator != AllocatorCPUManager && *c.Allocator != AllocatorExternal {
		return fmt.Errorf("invalid allocator %q: must be %q or %q",
			*c.Allocator, AllocatorCPUManager, AllocatorExternal)
	}
	return nil
}

// ValidateKubeletRootDir checks the flag-only path without adding it to the wire schema.
func ValidateKubeletRootDir(path string) error {
	// Socket locations and hostPath mounts must agree; silently defaulting an
	// empty value can register the plugin where the kubelet is not listening.
	if path == "" {
		return fmt.Errorf("invalid kubeletRootDir: must not be empty")
	}
	if !filepath.IsAbs(path) {
		return fmt.Errorf("invalid kubeletRootDir %q: must be an absolute path", path)
	}
	return nil
}
