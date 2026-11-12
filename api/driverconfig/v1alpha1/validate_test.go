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

func TestValidate(t *testing.T) {
	for _, tc := range []struct {
		name    string
		data    string
		wantErr string
	}{
		{name: "omitted fields", data: `{}`},
		{name: "null fields", data: `{"cpuDeviceMode":null,"groupBy":null,"allocator":null}`},
		{name: "grouped NUMA node CPU manager", data: `{"cpuDeviceMode":"grouped","groupBy":"numanode","allocator":"cpumanager"}`},
		{name: "individual socket external", data: `{"cpuDeviceMode":"individual","groupBy":"socket","allocator":"external"}`},
		{name: "machine external", data: `{"groupBy":"machine","allocator":"external"}`},
		{name: "cross-field constraints deferred", data: `{"cpuDeviceMode":"grouped","groupBy":"machine","allocator":"cpumanager"}`},
		{name: "unknown mode", data: `{"cpuDeviceMode":"unknown"}`, wantErr: `invalid cpuDeviceMode "unknown"`},
		{name: "empty mode", data: `{"cpuDeviceMode":""}`, wantErr: `invalid cpuDeviceMode ""`},
		{name: "unknown grouping", data: `{"groupBy":"unknown"}`, wantErr: `invalid groupBy "unknown"`},
		{name: "empty grouping", data: `{"groupBy":""}`, wantErr: `invalid groupBy ""`},
		{name: "unknown grouping in individual mode", data: `{"cpuDeviceMode":"individual","groupBy":"unknown"}`, wantErr: `invalid groupBy "unknown"`},
		{name: "unknown allocator", data: `{"kubeconfig":"/replacement","allocator":"unknown"}`, wantErr: `invalid allocator "unknown"`},
		{name: "empty allocator", data: `{"allocator":""}`, wantErr: `invalid allocator ""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var in v1alpha1.Config
			require.NoError(t, json.Unmarshal([]byte(tc.data), &in))
			baseline := driverconfig.Config{Kubeconfig: "/earlier"}
			out := baseline
			err := in.Validate()
			conversionErr := v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(&in, &out)
			if tc.wantErr == "" {
				require.NoError(t, err)
				require.NoError(t, conversionErr)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
				require.ErrorContains(t, conversionErr, tc.wantErr)
				require.Equal(t, baseline, out, "failed conversion must not partially apply fields")
			}
		})
	}
}

func TestValidateKubeletRootDir(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "standard root", path: "/var/lib/kubelet"},
		{name: "relocated root", path: "/mnt/kubelet"},
		{name: "noncanonical absolute root", path: "/mnt/a/../kubelet//"},
		{name: "empty", wantErr: "invalid kubeletRootDir: must not be empty"},
		{name: "relative", path: "relative/kubelet", wantErr: `invalid kubeletRootDir "relative/kubelet": must be an absolute path`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := v1alpha1.ValidateKubeletRootDir(tc.path)
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.EqualError(t, err, tc.wantErr)
			}
		})
	}
}
