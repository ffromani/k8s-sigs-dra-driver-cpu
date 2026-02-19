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
	"fmt"
	"slices"
	"sort"

	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/cpuinfo"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"
)

// IndividualCoreManager manages Device objects based on the CPU topology.
// It groups CPUs by physical core to assign consecutive device IDs to hyperthreads.
// This allows the DRA scheduler, which requests resources in contiguous blocks,
// to co-locate workloads on hyperthreads of the same core.
type IndividualCoreManager struct {
	driverName        string
	cpuTopology       *cpuinfo.CPUTopology
	reservedCPUs      cpuset.CPUSet
	deviceNameToCPUID map[string]int
}

func NewIndividualCoreManager(name string, topo *cpuinfo.CPUTopology, resv cpuset.CPUSet) *IndividualCoreManager {
	return &IndividualCoreManager{
		driverName:        name,
		cpuTopology:       topo,
		reservedCPUs:      resv,
		deviceNameToCPUID: make(map[string]int),
	}
}

func (mgr *IndividualCoreManager) CreateSlices(_ klog.Logger) [][]resourceapi.Device {
	reservedCPUs := make(map[int]bool)
	for _, cpuID := range mgr.reservedCPUs.List() {
		reservedCPUs[cpuID] = true
	}

	var availableCPUs []cpuinfo.CPUInfo
	topo := mgr.cpuTopology
	allCPUs := make([]cpuinfo.CPUInfo, 0, len(topo.CPUDetails))
	for _, cpu := range topo.CPUDetails {
		allCPUs = append(allCPUs, cpu)
		if !reservedCPUs[cpu.CpuID] {
			availableCPUs = append(availableCPUs, cpu)
		}
	}
	sort.Slice(availableCPUs, func(i, j int) bool {
		return availableCPUs[i].CpuID < availableCPUs[j].CpuID
	})

	processedCpus := make(map[int]bool)
	var coreGroups [][]cpuinfo.CPUInfo
	cpuInfoMap := make(map[int]cpuinfo.CPUInfo)
	for _, info := range allCPUs {
		cpuInfoMap[info.CpuID] = info
	}

	for _, cpu := range availableCPUs {
		if processedCpus[cpu.CpuID] {
			continue
		}
		if cpu.SiblingCpuID == -1 || reservedCPUs[cpu.SiblingCpuID] {
			coreGroups = append(coreGroups, []cpuinfo.CPUInfo{cpu})
			processedCpus[cpu.CpuID] = true
		} else {
			coreGroups = append(coreGroups, []cpuinfo.CPUInfo{cpu, cpuInfoMap[cpu.SiblingCpuID]})
			processedCpus[cpu.CpuID] = true
			processedCpus[cpu.SiblingCpuID] = true
		}
	}

	sort.Slice(coreGroups, func(i, j int) bool {
		return coreGroups[i][0].CpuID < coreGroups[j][0].CpuID
	})

	devId := 0
	var allDevices []resourceapi.Device
	for _, group := range coreGroups {
		for _, cpu := range group {
			deviceName := fmt.Sprintf("%s%03d", cpuDevicePrefix, devId)
			devId++
			mgr.deviceNameToCPUID[deviceName] = cpu.CpuID
			cpuDevice := resourceapi.Device{
				Name:       deviceName,
				Attributes: MakeIndividualAttributes(cpu),
				Capacity:   make(map[resourceapi.QualifiedName]resourceapi.DeviceCapacity),
			}
			allDevices = append(allDevices, cpuDevice)
		}
	}

	if len(allDevices) == 0 {
		return nil
	}

	// Chunk devices into slices of at most maxDevicesPerResourceSlice
	return slices.Collect(slices.Chunk(allDevices, maxDevicesPerResourceSlice))
}

func (mgr *IndividualCoreManager) AllocateCPUs(logger klog.Logger, claim *resourceapi.ResourceClaim) (cpuset.CPUSet, error) {
	logger = klog.LoggerWithValues(logger, "claim", claim.Namespace+"/"+claim.Name)

	claimCPUIDs := []int{}

	for _, alloc := range claim.Status.Allocation.Devices.Results {
		if alloc.Driver != mgr.driverName {
			continue
		}
		cpuID, ok := mgr.deviceNameToCPUID[alloc.Device]
		if !ok {
			return cpuset.CPUSet{}, fmt.Errorf("device %q not found in device to CPU ID map", alloc.Device)
		}
		claimCPUIDs = append(claimCPUIDs, cpuID)
	}

	if len(claimCPUIDs) == 0 {
		logger.V(5).Info("AllocateCPUs no CPU allocations for this driver")
		return cpuset.CPUSet{}, nil
	}

	return cpuset.New(claimCPUIDs...), nil
}
