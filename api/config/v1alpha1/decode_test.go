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

import "testing"

func TestDecodePreservesPresence(t *testing.T) {
	cfg, err := Decode([]byte("apiVersion: v1alpha1\nkubeconfig: ''\npublishNodeAllocatableResourceMapping: false\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CPUDeviceMode != nil || cfg.GroupBy != nil || cfg.Allocator != nil {
		t.Fatal("decoding an overlay must not fill omitted fields with defaults")
	}
	if cfg.Kubeconfig == nil || *cfg.Kubeconfig != "" {
		t.Fatal("explicit empty kubeconfig must retain its presence")
	}
	if cfg.PublishNodeAllocatableResourceMapping == nil || *cfg.PublishNodeAllocatableResourceMapping {
		t.Fatal("explicit false must retain its presence")
	}
}
