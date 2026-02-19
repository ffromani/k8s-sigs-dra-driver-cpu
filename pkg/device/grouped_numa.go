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

	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/cpuinfo"
	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/cpumanager"
	resourceapi "k8s.io/api/resource/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/klog/v2"
	"k8s.io/utils/cpuset"
	"k8s.io/utils/ptr"
)

type NUMAGroupedManager struct {
	driverName             string
	cpuTopology            *cpuinfo.CPUTopology
	reservedCPUs           cpuset.CPUSet
	getSharedCPUs          func() cpuset.CPUSet
	deviceNameToNUMANodeID map[string]int
}

func NewNUMAGroupedManager(name string, topo *cpuinfo.CPUTopology, resv cpuset.CPUSet, getSharedCPUs func() cpuset.CPUSet) *NUMAGroupedManager {
	return &NUMAGroupedManager{
		driverName:             name,
		cpuTopology:            topo,
		reservedCPUs:           resv,
		getSharedCPUs:          getSharedCPUs,
		deviceNameToNUMANodeID: make(map[string]int),
	}
}

func (mgr *NUMAGroupedManager) CreateSlices(_ klog.Logger) [][]resourceapi.Device {
	klog.Info("Creating grouped CPU devices", "groupBy", "NUMANode")
	var devices []resourceapi.Device

	numaNodeIDs := mgr.cpuTopology.CPUDetails.NUMANodes().List()
	for _, numaIDInt := range numaNodeIDs {
		numaID := int64(numaIDInt)
		deviceName := fmt.Sprintf("%s%03d", cpuDeviceNUMAGroupedPrefix, numaIDInt)
		numaNodeCPUSet := mgr.cpuTopology.CPUDetails.CPUsInNUMANodes(numaIDInt)
		allocatableCPUs := numaNodeCPUSet.Difference(mgr.reservedCPUs)
		availableCPUsInNUMANode := int64(allocatableCPUs.Size())

		if allocatableCPUs.Size() == 0 {
			continue
		}

		// All CPUs in a NUMA node belong to the same socket.
		anyCPU := allocatableCPUs.UnsortedList()[0]
		socketID := int64(mgr.cpuTopology.CPUDetails[anyCPU].SocketID)

		deviceCapacity := map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
			cpuResourceQualifiedName: {Value: *resource.NewQuantity(availableCPUsInNUMANode, resource.DecimalSI)},
		}

		mgr.deviceNameToNUMANodeID[deviceName] = numaIDInt

		deviceAttributes := MakeGroupedAttributes(mgr.cpuTopology, socketID, allocatableCPUs)
		deviceAttributes["dra.cpu/numaNodeID"] = resourceapi.DeviceAttribute{IntValue: &numaID}
		// TODO(pravk03): Remove. Hack to align with NIC (DRANet). We need some standard attribute to align other resources with CPU.
		deviceAttributes["dra.net/numaNode"] = resourceapi.DeviceAttribute{IntValue: &numaID}

		devices = append(devices, resourceapi.Device{
			Name:                     deviceName,
			Attributes:               deviceAttributes,
			Capacity:                 deviceCapacity,
			AllowMultipleAllocations: ptr.To(true),
		})
	}

	if len(devices) == 0 {
		return nil
	}
	return [][]resourceapi.Device{devices}
}

func (mgr *NUMAGroupedManager) AllocateCPUs(logger klog.Logger, claim *resourceapi.ResourceClaim) (cpuset.CPUSet, error) {
	logger = klog.LoggerWithValues(logger, "claim", claim.Namespace+"/"+claim.Name)

	var cpuAssignment cpuset.CPUSet

	for _, alloc := range claim.Status.Allocation.Devices.Results {
		claimCPUCount := int64(0)
		if alloc.Driver != mgr.driverName {
			continue
		}
		if quantity, ok := alloc.ConsumedCapacity[cpuResourceQualifiedName]; ok {
			count := quantity.Value()
			claimCPUCount = count
			logger.Info("Found CPUs request", "CPUCount", count, "device", alloc.Device)
		}

		var availableCPUsForDevice cpuset.CPUSet
		numaNodeID, ok := mgr.deviceNameToNUMANodeID[alloc.Device]
		if !ok {
			return cpuset.CPUSet{}, fmt.Errorf("no valid NUMA node ID found for device %s", alloc.Device)
		}
		numaCPUs := mgr.cpuTopology.CPUDetails.CPUsInNUMANodes(numaNodeID)
		availableCPUsForDevice = mgr.getSharedCPUs().Intersection(numaCPUs)
		logger.Info("available CPUs", "NUMANode", numaNodeID, "totalCPUs", numaCPUs.String(), "availableCPUs", availableCPUsForDevice.String())

		cur, err := cpumanager.TakeByTopologyNUMAPacked(logger, mgr.cpuTopology, availableCPUsForDevice, int(claimCPUCount), cpumanager.CPUSortingStrategyPacked, true)
		if err != nil {
			return cpuset.CPUSet{}, err
		}
		cpuAssignment = cpuAssignment.Union(cur)
		logger.Info("CPU assignment", "device", alloc.Device, "partialCPUs", cur.String(), "totalCPUs", cpuAssignment.String())
	}

	if cpuAssignment.Size() == 0 {
		logger.V(5).Info("AllocateCPUs no CPU allocations for this driver")
		return cpuset.CPUSet{}, nil
	}

	return cpuAssignment, nil
}
