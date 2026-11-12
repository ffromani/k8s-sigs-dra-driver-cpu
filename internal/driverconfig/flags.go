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
	"flag"

	"github.com/go-logr/logr"
)

// flagToJSONKey maps CLI flag names to their Config JSON keys.
// Add an entry here whenever a new field is added to Config and AddFlags.
var flagToJSONKey = map[string]string{
	"kubeconfig":        "kubeconfig",
	"bind-address":      "bindAddress",
	"expose-pcie-roots": "exposePCIeRoots",
	"kubelet-root-dir":  "kubeletRootDir",
}

type FlagSource struct {
	flagset         *flag.FlagSet
	kubeConfig      string
	bindAddress     string
	kubeletRootDir  string
	exposePCIeRoots bool
}

func FromFlags(flagset *flag.FlagSet) FlagSource {
	fs := FlagSource{
		flagset:        flagset,
		bindAddress:    DefaultBindAddress,
		kubeletRootDir: DefaultKubeletRootDir,
	}
	flagset.StringVar(&fs.kubeConfig, "kubeconfig", fs.kubeConfig, "absolute path to the kubeconfig file")
	flagset.StringVar(&fs.bindAddress, "bind-address", fs.bindAddress, "The address to bind the HTTP server for /healthz and /metrics endpoints")
	flagset.BoolVar(&fs.exposePCIeRoots, "expose-pcie-roots", fs.exposePCIeRoots, "Discover and expose PCIe roots as device attributes. Requires the DRAListTypeAttributes=true Feature Gate in the cluster.")
	flagset.StringVar(&fs.kubeletRootDir, "kubelet-root-dir", fs.kubeletRootDir, "The kubelet root directory. The plugin registration and plugin data directories are derived from it as <root>/plugins_registry and <root>/plugins/<driver-name>. The Helm chart supplies this together with the matching hostPath mounts; set it only if the kubelet --root-dir is not the default /var/lib/kubelet.")
	return fs
}

func (fs FlagSource) Name() string {
	return "flags"
}

func (fs FlagSource) Apply(logger logr.Logger, cfg *Config) error {
	overrides := make(map[string]any)
	fs.flagset.Visit(func(f *flag.Flag) {
		key, ok := flagToJSONKey[f.Name]
		if !ok {
			return
		}
		if g, ok := f.Value.(flag.Getter); ok {
			overrides[key] = g.Get()
		} else {
			overrides[key] = f.Value.String()
		}
	})
	logger.V(6).Info("overrides", "stage", fs.Name(), "values", overrides)
	return applyMap(cfg, overrides)
}
