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

	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig"
)

// Convert_v1alpha1_Config_To_driverconfig_Config applies only supplied fields,
// so configlets cannot reset settings established by earlier sources.
func Convert_v1alpha1_Config_To_driverconfig_Config(in *Config, out *driverconfig.Config) error {
	if in == nil {
		return nil
	}
	if err := in.Validate(); err != nil {
		return fmt.Errorf("validate %s configuration: %w", APIVersion, err)
	}
	if in.Kubeconfig != nil {
		out.Kubeconfig = *in.Kubeconfig
	}
	if in.HostnameOverride != nil {
		out.HostnameOverride = *in.HostnameOverride
	}
	if in.ReservedCPUs != nil {
		out.ReservedCPUs = *in.ReservedCPUs
	}
	if in.CPUDeviceMode != nil {
		out.CPUDeviceMode = string(*in.CPUDeviceMode)
	}
	if in.GroupBy != nil {
		out.GroupBy = string(*in.GroupBy)
	}
	if in.SysFSOverlay != nil {
		out.SysFSOverlay = *in.SysFSOverlay
	}
	if in.PublishNodeAllocatableResourceMapping != nil {
		out.PublishNodeAllocatableResourceMapping = *in.PublishNodeAllocatableResourceMapping
	}
	if in.Allocator != nil {
		out.Allocator = string(*in.Allocator)
	}
	return nil
}
