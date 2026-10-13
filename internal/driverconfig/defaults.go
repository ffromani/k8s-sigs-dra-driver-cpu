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

import "github.com/kubernetes-sigs/dra-driver-cpu/api/config/v1alpha1"

// CLI-only deployment defaults are independent of the public file schema.
const (
	DefaultKubeletRootDir = "/var/lib/kubelet"
	DefaultBindAddress    = ":8080"
)

// Default resolves an empty configuration of the current public version.
func Default() Config {
	cfg := Config{
		BindAddress:    DefaultBindAddress,
		KubeletRootDir: DefaultKubeletRootDir,
	}
	FromV1Alpha1(v1alpha1.Default(), &cfg)
	return cfg
}
