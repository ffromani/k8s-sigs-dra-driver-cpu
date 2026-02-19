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

type SocketGroupedManager struct {
	driverName           string
	cpuTopology          *cpuinfo.CPUTopology
	reservedCPUs         cpuset.CPUSet
	getSharedCPUs        func() cpuset.CPUSet
	deviceNameToSocketID map[string]int
}

func NewSocketGroupedManager(name string, topo *cpuinfo.CPUTopology, resv cpuset.CPUSet, getSharedCPUs func() cpuset.CPUSet) *SocketGroupedManager {
	return &SocketGroupedManager{
		driverName:           name,
		cpuTopology:          topo,
		reservedCPUs:         resv,
		getSharedCPUs:        getSharedCPUs,
		deviceNameToSocketID: make(map[string]int),
	}
}

func (mgr *SocketGroupedManager) CreateSlices(logger klog.Logger) [][]resourceapi.Device {
	logger.Info("Creating grouped CPU devices", "groupBy", "Socket")
	var devices []resourceapi.Device

	socketIDs := mgr.cpuTopology.CPUDetails.Sockets().List()
	for _, socketIDInt := range socketIDs {
		socketID := int64(socketIDInt)
		deviceName := fmt.Sprintf("%s%03d", cpuDeviceSocketGroupedPrefix, socketIDInt)
		socketCPUSet := mgr.cpuTopology.CPUDetails.CPUsInSockets(socketIDInt)
		allocatableCPUs := socketCPUSet.Difference(mgr.reservedCPUs)
		availableCPUsInSocket := int64(allocatableCPUs.Size())

		if allocatableCPUs.Size() == 0 {
			continue
		}

		deviceCapacity := map[resourceapi.QualifiedName]resourceapi.DeviceCapacity{
			cpuResourceQualifiedName: {Value: *resource.NewQuantity(availableCPUsInSocket, resource.DecimalSI)},
		}

		mgr.deviceNameToSocketID[deviceName] = socketIDInt

		devices = append(devices, resourceapi.Device{
			Name:                     deviceName,
			Attributes:               MakeGroupedAttributes(mgr.cpuTopology, socketID, allocatableCPUs),
			Capacity:                 deviceCapacity,
			AllowMultipleAllocations: ptr.To(true),
		})
	}

	if len(devices) == 0 {
		return nil
	}
	return [][]resourceapi.Device{devices}
}

func (mgr *SocketGroupedManager) AllocateCPUs(logger klog.Logger, claim *resourceapi.ResourceClaim) (cpuset.CPUSet, error) {
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
		socketID, ok := mgr.deviceNameToSocketID[alloc.Device]
		if !ok {
			return cpuset.CPUSet{}, fmt.Errorf("no valid socket ID found for device %s", alloc.Device)
		}
		socketCPUs := mgr.cpuTopology.CPUDetails.CPUsInSockets(socketID)
		availableCPUsForDevice = mgr.getSharedCPUs().Intersection(socketCPUs)
		logger.Info("available CPUs", "Socket", socketID, "totalCPUs", socketCPUs.String(), "availableCPUs", availableCPUsForDevice.String())

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
