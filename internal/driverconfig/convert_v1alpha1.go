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
	"github.com/go-logr/logr"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/config/v1alpha1"
)

type V1Alpha1Source struct {
	Config v1alpha1.Config
}

func (src V1Alpha1Source) Name() string {
	return "v1alpha1"
}

func (src V1Alpha1Source) Apply(_ logr.Logger, cfg *Config) error {
	FromV1Alpha1(src.Config, cfg)
	return nil
}

func (src V1Alpha1Source) String() string {
	return src.Config.String()
}

// FromV1Alpha1 overlays explicitly supplied public fields onto runtime configuration.
func FromV1Alpha1(in v1alpha1.Config, cfg *Config) {
	if in.Kubeconfig != nil {
		cfg.Kubeconfig = *in.Kubeconfig
	}
	if in.HostnameOverride != nil {
		cfg.HostnameOverride = *in.HostnameOverride
	}
	if in.ReservedCPUs != nil {
		cfg.ReservedCPUs = *in.ReservedCPUs
	}
	if in.CPUDeviceMode != nil {
		cfg.CPUDeviceMode = *in.CPUDeviceMode
	}
	if in.GroupBy != nil {
		cfg.GroupBy = *in.GroupBy
	}
	if in.SysFSOverlay != nil {
		cfg.SysFSOverlay = *in.SysFSOverlay
	}
	if in.PublishNodeAllocatableResourceMapping != nil {
		cfg.PublishNodeAllocatableResourceMapping = *in.PublishNodeAllocatableResourceMapping
	}
	if in.Allocator != nil {
		cfg.Allocator = *in.Allocator
	}
}

// ToV1Alpha1 exports the resolved public fields, including explicit zero values.
func ToV1Alpha1(in Config) v1alpha1.Config {
	return v1alpha1.Config{
		APIVersion:                            v1alpha1.APIVersion,
		Kubeconfig:                            new(in.Kubeconfig),
		HostnameOverride:                      new(in.HostnameOverride),
		ReservedCPUs:                          new(in.ReservedCPUs),
		CPUDeviceMode:                         new(in.CPUDeviceMode),
		GroupBy:                               new(in.GroupBy),
		SysFSOverlay:                          new(in.SysFSOverlay),
		PublishNodeAllocatableResourceMapping: new(in.PublishNodeAllocatableResourceMapping),
		Allocator:                             new(in.Allocator),
	}
}
