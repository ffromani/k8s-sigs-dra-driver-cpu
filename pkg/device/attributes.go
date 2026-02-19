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

import (
	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/cpuinfo"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/utils/cpuset"
)

func MakeIndividualAttributes(cpu cpuinfo.CPUInfo) map[resourceapi.QualifiedName]resourceapi.DeviceAttribute {
	numaNode := int64(cpu.NUMANodeID)
	cacheL3ID := int64(cpu.UncoreCacheID)
	socketID := int64(cpu.SocketID)
	coreID := int64(cpu.CoreID)
	cpuID := int64(cpu.CpuID)
	coreType := cpu.CoreType.String()
	return map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
		"dra.cpu/numaNodeID": {IntValue: &numaNode},
		"dra.cpu/cacheL3ID":  {IntValue: &cacheL3ID},
		"dra.cpu/coreType":   {StringValue: &coreType},
		"dra.cpu/socketID":   {IntValue: &socketID},
		"dra.cpu/coreID":     {IntValue: &coreID},
		"dra.cpu/cpuID":      {IntValue: &cpuID},
		// TODO(pravk03): Remove. Hack to align with NIC (DRANet). We need some standard attribute to align other resources with CPU.
		"dra.net/numaNode": {IntValue: &numaNode},
	}
}

func MakeGroupedAttributes(topo *cpuinfo.CPUTopology, socketID int64, allocatableCPUs cpuset.CPUSet) map[resourceapi.QualifiedName]resourceapi.DeviceAttribute {
	smtEnabled := topo.SMTEnabled
	availableCPUs := int64(allocatableCPUs.Size())
	return map[resourceapi.QualifiedName]resourceapi.DeviceAttribute{
		"dra.cpu/socketID":   {IntValue: &socketID},
		"dra.cpu/numCPUs":    {IntValue: &availableCPUs},
		"dra.cpu/smtEnabled": {BoolValue: &smtEnabled},
	}
}
