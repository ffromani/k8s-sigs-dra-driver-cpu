/*
Copyright 2026 The Kubernetes Authors.

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

package device

const (
	// maxDevicesPerResourceSlice is the maximum number of devices that can be packed into a single
	// ResourceSlice object. This is a hard limit defined in the Kubernetes API at
	// https://github.com/kubernetes/kubernetes/blob/8e6d788887034b799f6c2a86991a68a080bb0576/pkg/apis/resource/types.go#L245
	maxDevicesPerResourceSlice = 128
	cpuDevicePrefix            = "cpudev"

	// Grouped Mode
	// cpuResourceQualifiedName is the qualified name for the CPU resource capacity.
	cpuResourceQualifiedName = "dra.cpu/cpu"

	cpuDeviceSocketGroupedPrefix = "cpudevsocket"
	cpuDeviceNUMAGroupedPrefix   = "cpudevnuma"
)
