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
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfigLogValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  Config
		want map[string]any
	}{
		{
			name: "zero values remain visible",
			want: map[string]any{
				"kubeconfig": "", "hostnameOverride": "", "bindAddress": "",
				"reservedCPUs": "", "cpuDeviceMode": "", "groupBy": "",
				"exposePCIeRoots": false, "sysfsOverlay": "", "kubeletRootDir": "",
				"publishNodeAllocatableResourceMapping": false, "allocator": "",
			},
		},
		{
			name: "effective settings include CLI-only fields",
			cfg: Config{
				Kubeconfig: "/client/config", HostnameOverride: "worker-1", BindAddress: ":9090",
				ReservedCPUs: "0-3", CPUDeviceMode: "grouped", GroupBy: "socket",
				ExposePCIeRoots: true, SysFSOverlay: "/topology.yaml", KubeletRootDir: "/custom/kubelet",
				PublishNodeAllocatableResourceMapping: true, Allocator: "external",
			},
			want: map[string]any{
				"kubeconfig": "/client/config", "hostnameOverride": "worker-1", "bindAddress": ":9090",
				"reservedCPUs": "0-3", "cpuDeviceMode": "grouped", "groupBy": "socket",
				"exposePCIeRoots": true, "sysfsOverlay": "/topology.yaml", "kubeletRootDir": "/custom/kubelet",
				"publishNodeAllocatableResourceMapping": true, "allocator": "external",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.cfg.LogValues()
			require.Len(t, values, 2*reflect.TypeFor[Config]().NumField())
			got := make(map[string]any)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				require.True(t, ok, "log key at index %d is not a string", i)
				require.NotContains(t, got, key, "duplicate log key")
				got[key] = values[i+1]
			}
			require.Equal(t, tc.want, got)
		})
	}
}
