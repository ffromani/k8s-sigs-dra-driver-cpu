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

package v1alpha1_test

import (
	"encoding/json"
	"testing"

	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestConvertPartialConfig(t *testing.T) {
	baseline := driverconfig.Config{
		Kubeconfig: "/earlier/kubeconfig", HostnameOverride: "earlier-node",
		ReservedCPUs: "0-3", CPUDeviceMode: "grouped", GroupBy: "socket",
		SysFSOverlay: "/earlier/sysfs", PublishNodeAllocatableResourceMapping: true,
		Allocator: "external", BindAddress: ":9090", KubeletRootDir: "/custom/kubelet",
		ExposePCIeRoots: true,
	}
	cleared := driverconfig.Config{
		BindAddress: ":9090", KubeletRootDir: "/custom/kubelet", ExposePCIeRoots: true,
		CPUDeviceMode: "grouped", GroupBy: "socket", Allocator: "external",
	}
	partial := baseline
	partial.ReservedCPUs = "4-7"
	for _, tc := range []struct {
		name string
		data string
		want driverconfig.Config
	}{
		{name: "omitted fields preserve earlier settings", data: `{}`, want: baseline},
		{
			name: "legacy null preserves earlier settings",
			data: `{"kubeconfig":null,"hostnameOverride":null,"reservedCPUs":null,
				"cpuDeviceMode":null,"groupBy":null,"sysfsOverlay":null,
				"publishNodeAllocatableResourceMapping":null,"allocator":null}`,
			want: baseline,
		},
		{
			name: "explicit empty and false override earlier settings",
			data: `{"kubeconfig":"","hostnameOverride":"","reservedCPUs":"",
				"sysfsOverlay":"",
				"publishNodeAllocatableResourceMapping":false}`,
			want: cleared,
		},
		{name: "partial config preserves unrelated settings", data: `{"reservedCPUs":"4-7"}`, want: partial},
		{
			name: "all supplied fields are converted",
			data: `{"kubeconfig":"/new/kubeconfig","hostnameOverride":"new-node","reservedCPUs":"8-11",
				"cpuDeviceMode":"individual","groupBy":"numanode","sysfsOverlay":"/new/sysfs",
				"publishNodeAllocatableResourceMapping":false,"allocator":"cpumanager"}`,
			want: driverconfig.Config{
				Kubeconfig: "/new/kubeconfig", HostnameOverride: "new-node", ReservedCPUs: "8-11",
				CPUDeviceMode: "individual", GroupBy: "numanode", SysFSOverlay: "/new/sysfs",
				Allocator: "cpumanager", BindAddress: ":9090", KubeletRootDir: "/custom/kubelet",
				ExposePCIeRoots: true,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var in v1alpha1.Config
			require.NoError(t, json.Unmarshal([]byte(tc.data), &in))
			got := baseline
			require.NoError(t, v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(&in, &got))
			require.Equal(t, tc.want, got)
		})
	}
}

func TestConvertDefaultConfig(t *testing.T) {
	defaults := v1alpha1.Default()
	var got driverconfig.Config
	require.NoError(t, v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(&defaults, &got))
	require.Equal(t, driverconfig.Config{
		CPUDeviceMode: "grouped", GroupBy: "numanode", Allocator: "cpumanager",
	}, got)
}

func TestConfigJSONPresence(t *testing.T) {
	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "omitted", data: `{}`},
		{name: "explicit zero", data: `{"reservedCPUs":"","publishNodeAllocatableResourceMapping":false}`},
		{name: "explicit values", data: `{"reservedCPUs":"4-7","publishNodeAllocatableResourceMapping":true}`},
		{name: "enum values", data: `{"cpuDeviceMode":"grouped","groupBy":"machine","allocator":"external"}`},
		{name: "unknown enum values survive unmarshaling for conversion validation", data: `{"cpuDeviceMode":"unknown","groupBy":"unknown","allocator":"unknown"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var cfg v1alpha1.Config
			require.NoError(t, json.Unmarshal([]byte(tc.data), &cfg))
			out, err := json.Marshal(cfg)
			require.NoError(t, err)
			require.JSONEq(t, tc.data, string(out))
		})
	}
}

func TestSchemaPreservesLegacyWireTypes(t *testing.T) {
	out, err := v1alpha1.GenerateDriverConfigSchema()
	require.NoError(t, err)
	var schema struct {
		Properties map[string]struct {
			Type string   `json:"type"`
			Enum []string `json:"enum"`
		} `json:"properties"`
		Required []string `json:"required"`
	}
	require.NoError(t, json.Unmarshal(out, &schema))
	for _, key := range []string{
		"kubeconfig", "hostnameOverride", "reservedCPUs", "cpuDeviceMode",
		"groupBy", "sysfsOverlay", "allocator", "apiVersion",
	} {
		require.Equal(t, "string", schema.Properties[key].Type, key)
	}
	require.Equal(t, "boolean", schema.Properties["publishNodeAllocatableResourceMapping"].Type)
	require.Empty(t, schema.Required)
	for key, want := range map[string][]string{
		"cpuDeviceMode": {"grouped", "individual"},
		"groupBy":       {"numanode", "socket", "machine"},
		"allocator":     {"cpumanager", "external"},
	} {
		require.Equal(t, want, schema.Properties[key].Enum, key)
	}
}

func TestConvertEnums(t *testing.T) {
	for _, tc := range []struct {
		name      string
		mode      v1alpha1.CPUDeviceMode
		group     v1alpha1.GroupBy
		allocator v1alpha1.Allocator
		want      driverconfig.Config
	}{
		{
			name: "grouped NUMA node with CPU manager",
			mode: v1alpha1.CPUDeviceModeGrouped, group: v1alpha1.GroupByNUMANode,
			allocator: v1alpha1.AllocatorCPUManager,
			want:      driverconfig.Config{CPUDeviceMode: "grouped", GroupBy: "numanode", Allocator: "cpumanager"},
		},
		{
			name: "individual socket with external allocator",
			mode: v1alpha1.CPUDeviceModeIndividual, group: v1alpha1.GroupBySocket,
			allocator: v1alpha1.AllocatorExternal,
			want:      driverconfig.Config{CPUDeviceMode: "individual", GroupBy: "socket", Allocator: "external"},
		},
		{
			name: "grouped machine with external allocator",
			mode: v1alpha1.CPUDeviceModeGrouped, group: v1alpha1.GroupByMachine,
			allocator: v1alpha1.AllocatorExternal,
			want:      driverconfig.Config{CPUDeviceMode: "grouped", GroupBy: "machine", Allocator: "external"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := v1alpha1.Config{CPUDeviceMode: &tc.mode, GroupBy: &tc.group, Allocator: &tc.allocator}
			var got driverconfig.Config
			require.NoError(t, v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(&in, &got))
			require.Equal(t, tc.want, got)
		})
	}
}
