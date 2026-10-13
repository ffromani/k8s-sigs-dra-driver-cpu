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
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
)

type FlagSource struct {
	cliopts CLIOptions
}

// FromFlags snapshots explicitly supplied options after parsing.
func FromFlags(cliopts CLIOptions) FlagSource {
	return FlagSource{
		cliopts: cliopts,
	}
}

func (fs FlagSource) Name() string {
	return "flags"
}

func (fs FlagSource) String() string {
	return fs.cliopts.String()
}

func (fs FlagSource) Apply(logger logr.Logger, cfg *Config) error {
	logger.V(6).Info("overrides", "stage", fs.Name(), "values", fs.String())
	cliopts := fs.cliopts
	if cliopts.Kubeconfig.Set {
		cfg.Kubeconfig = cliopts.Kubeconfig.Value
	}
	if cliopts.BindAddress.Set {
		cfg.BindAddress = cliopts.BindAddress.Value
	}
	if cliopts.ExposePCIeRoots.Set {
		cfg.ExposePCIeRoots = cliopts.ExposePCIeRoots.Value
	}
	if cliopts.KubeletRootDir.Set {
		cfg.KubeletRootDir = cliopts.KubeletRootDir.Value
	}
	return nil
}

// CLIOption holds a parsed value and whether it was explicitly supplied.
type CLIOption[T any] struct {
	Value T
	Set   bool
}

type CLIOptions struct {
	Kubeconfig      CLIOption[string]
	BindAddress     CLIOption[string]
	ExposePCIeRoots CLIOption[bool]
	KubeletRootDir  CLIOption[string]
}

func DefaultCLIOptions() CLIOptions {
	c := CLIOptions{}
	c.SetDefaults()
	return c
}

func (cliopts CLIOptions) String() string {
	var sb strings.Builder
	if cliopts.Kubeconfig.Set {
		sb.WriteString("kubeConfig=" + cliopts.Kubeconfig.Value + " ")
	}
	if cliopts.BindAddress.Set {
		sb.WriteString("bindAddress=" + cliopts.BindAddress.Value + " ")
	}
	if cliopts.ExposePCIeRoots.Set {
		sb.WriteString("exposePCIeRoots=" + strconv.FormatBool(cliopts.ExposePCIeRoots.Value) + " ")
	}
	if cliopts.KubeletRootDir.Set {
		sb.WriteString("kubeletRootDir=" + cliopts.KubeletRootDir.Value + " ")
	}
	if sb.Len() == 0 {
		return "<none>"
	}
	return strings.TrimSuffix(sb.String(), " ")
}

func (c *CLIOptions) SetDefaults() {
	if !c.BindAddress.Set && c.BindAddress.Value == "" {
		c.BindAddress.Value = DefaultBindAddress
	}
	if !c.KubeletRootDir.Set && c.KubeletRootDir.Value == "" {
		c.KubeletRootDir.Value = DefaultKubeletRootDir
	}
}

func (c *CLIOptions) AddFlags(fs *flag.FlagSet) {
	c.SetDefaults()

	fs.Func("kubeconfig", "absolute path to the kubeconfig file (`string`)", func(value string) error {
		c.Kubeconfig = CLIOption[string]{Value: value, Set: true}
		return nil
	})
	fs.Func("bind-address", "The address to bind the HTTP server for /healthz and /metrics endpoints (`string`)", func(value string) error {
		c.BindAddress = CLIOption[string]{Value: value, Set: true}
		return nil
	})
	fs.BoolFunc("expose-pcie-roots", deprecatedUsage(
		"exposePCIeRoots",
		"Discover and expose PCIe roots as device attributes. Requires the DRAListTypeAttributes=true Feature Gate in the cluster."), func(value string) error {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		c.ExposePCIeRoots = CLIOption[bool]{Value: parsed, Set: true}
		return nil
	})
	fs.Func("kubelet-root-dir", "The kubelet root directory (`string`). The plugin registration and plugin data directories are derived from it as <root>/plugins_registry and <root>/plugins/<driver-name>. The Helm chart supplies this together with the matching hostPath mounts; set it only if the kubelet --root-dir is not the default /var/lib/kubelet.", func(value string) error {
		c.KubeletRootDir = CLIOption[string]{Value: value, Set: true}
		return nil
	})

	// Func flags do not infer defaults from their callbacks; retain them in help.
	fs.Lookup("kubeconfig").DefValue = c.Kubeconfig.Value
	fs.Lookup("bind-address").DefValue = c.BindAddress.Value
	fs.Lookup("kubelet-root-dir").DefValue = c.KubeletRootDir.Value
	if c.ExposePCIeRoots.Value {
		fs.Lookup("expose-pcie-roots").DefValue = "true"
	}
}

func deprecatedUsage(driverCLIOptionsField, usage string) string {
	return fmt.Sprintf("%s (DEPRECATED: prefer driverCLIOptions field %q; this flag will be removed in a future release)", usage, driverCLIOptionsField)
}

// WarnDeprecatedFlags warns only for explicitly supplied deprecated flags,
// including an explicit false value.
func WarnDeprecatedFlags(fs *flag.FlagSet, logger logr.Logger) {
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "expose-pcie-roots" {
			logger.Info("flag is deprecated and will be removed in a future release; replacement through versioned configuration is planned",
				"flag", f.Name)
		}
	})
}
