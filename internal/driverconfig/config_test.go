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

// The test file lives in package driverconfig_test (external test package) to
// verify the exported API without access to internal helpers.
package driverconfig_test

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr/funcr"
	"github.com/go-logr/logr/testr"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/config/v1alpha1"
	"github.com/kubernetes-sigs/dra-driver-cpu/internal/driverconfig"
	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/device"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFlagSet creates a FlagSet with cliopts registered and args parsed.
func newFlagSet(t *testing.T, cliopts *driverconfig.CLIOptions, args []string) *flag.FlagSet {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	cliopts.AddFlags(fs)
	require.NoError(t, fs.Parse(args))
	return fs
}

// writeFile creates name with content inside dir and returns the full path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	return path
}

// TestResolve_NoSources: no sources returns Default() unchanged.
func TestResolve_NoSources(t *testing.T) {
	result, err := driverconfig.Resolve(testr.New(t), nil)

	require.NoError(t, err)
	assert.Equal(t, driverconfig.Default(), result)
}

// TestResolve_FileOverridesDefaults: file values are applied when no CLI flags are set.
func TestResolve_FileOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
cpuDeviceMode: individual
groupBy: socket
reservedCPUs: "0-3"
sysfsOverlay: /custom/sysfs
publishNodeAllocatableResourceMapping: true
`)

	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.NoError(t, err)
	assert.Equal(t, device.CPU_DEVICE_MODE_INDIVIDUAL, result.CPUDeviceMode)
	assert.Equal(t, device.GROUP_BY_SOCKET, result.GroupBy)
	assert.Equal(t, "0-3", result.ReservedCPUs)
	assert.Equal(t, "/custom/sysfs", result.SysFSOverlay)
	assert.True(t, result.PublishNodeAllocatableResourceMapping)
}

// TestResolve_CLIFlagWinsOverFile: an explicitly-passed CLI flag beats the file value.
func TestResolve_CLIFlagWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
kubeconfig: /from-file
`)

	cliopts := driverconfig.DefaultCLIOptions()
	newFlagSet(t, &cliopts, []string{
		"--kubeconfig=/from-cli",
	})

	assert.Equal(t, "/from-cli", cliopts.Kubeconfig.Value)
	assert.True(t, cliopts.Kubeconfig.Set)

	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
		driverconfig.FromFlags(cliopts),
	})

	require.NoError(t, err)
	assert.Equal(t, "/from-cli", result.Kubeconfig)
}

// TestResolve_RejectsAmbiguousKeys checks case-sensitive field names, duplicate
// keys, and deployment-only fields. Incorrectly cased public fields are unknown;
// deployment-only fields retain hints about their supported alternatives.
func TestResolve_RejectsAmbiguousKeys(t *testing.T) {
	for _, tc := range []struct {
		name            string
		file            string
		flags           []string
		wantContains    []string
		wantNotContains []string
	}{{
		name:         "folded spelling",
		file:         "apiVersion: v1alpha1\nreservedcpus: \"0-3\"\n",
		wantContains: []string{`unknown field "reservedcpus"`},
	}, {
		name:         "folded spelling, mixed case",
		file:         "apiVersion: v1alpha1\nReservedCPUs: \"0-3\"\n",
		wantContains: []string{`unknown field "ReservedCPUs"`},
	}, {
		name:         "folded spelling, upper case",
		file:         "apiVersion: v1alpha1\nRESERVEDCPUS: \"0-3\"\n",
		wantContains: []string{`unknown field "RESERVEDCPUS"`},
	}, {
		name:         "folded spelling, one letter",
		file:         "apiVersion: v1alpha1\nreservedCpus: \"0-3\"\n",
		wantContains: []string{`unknown field "reservedCpus"`},
	}, {
		// Explicit flags must not make incorrectly cased file fields acceptable.
		name:         "folded spelling against an explicit flag",
		file:         "apiVersion: v1alpha1\nKubeconfig: \"2-3\"\n",
		flags:        []string{"--kubeconfig=/from-cli"},
		wantContains: []string{`unknown field "Kubeconfig"`},
	}, {
		name:         "the same key twice",
		file:         "apiVersion: v1alpha1\nreservedCPUs: \"0-1\"\nreservedCPUs: \"2-3\"\n",
		wantContains: []string{"reservedCPUs"},
	}, {
		name:         "the same key twice, folded",
		file:         "apiVersion: v1alpha1\nreservedCPUs: \"0-1\"\nreservedcpus: \"2-3\"\n",
		wantContains: []string{"reservedcpus"},
	}, {
		name:            "folded apiVersion",
		file:            "ApiVersion: v1alpha1\nreservedCPUs: \"0-1\"\n",
		wantContains:    []string{`use "apiVersion"`},
		wantNotContains: []string{"unknown field"},
	}, {
		name:            "folded excluded field",
		file:            "apiVersion: v1alpha1\nbindaddress: \":9999\"\n",
		wantContains:    []string{"not configurable via the config file", "healthzPort"},
		wantNotContains: []string{"is spelled differently"},
	}, {
		name:            "folded excluded field, upper case",
		file:            "apiVersion: v1alpha1\nBINDADDRESS: \":9999\"\n",
		wantContains:    []string{"not configurable via the config file", "healthzPort"},
		wantNotContains: []string{"is spelled differently"},
	}, {
		name:            "folded excluded field with a flag alternative",
		file:            "apiVersion: v1alpha1\nexposepcieroots: true\n",
		wantContains:    []string{"not configurable via the config file", "args.exposePCIeRoots"},
		wantNotContains: []string{"is spelled differently"},
	}, {
		name:            "a name that matches nothing",
		file:            "apiVersion: v1alpha1\ntotallyUnknownField: 1\n",
		wantContains:    []string{"totallyUnknownField"},
		wantNotContains: []string{"spelled differently"},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgFile := writeFile(t, dir, "config.yaml", tc.file)

			cliopts := driverconfig.DefaultCLIOptions()
			sources := []driverconfig.Source{driverconfig.FromFile(cfgFile)}
			if len(tc.flags) > 0 {
				newFlagSet(t, &cliopts, tc.flags)
				sources = append(sources, driverconfig.FromFlags(cliopts))
			}

			_, err := driverconfig.Resolve(testr.New(t), sources)

			require.Error(t, err)
			for _, want := range tc.wantContains {
				assert.Contains(t, err.Error(), want)
			}
			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, err.Error(), notWant)
			}
		})
	}
}

// TestResolve_ReportsEveryMiscasedKey checks that every key is reported in
// deterministic order, independently of error separators and source wrapping.
func TestResolve_ReportsEveryMiscasedKey(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
reservedcpus: "0-3"
groupby: socket
cpudevicemode: individual
`)

	_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.Error(t, err)
	err2 := err
	for cause := errors.Unwrap(err2); cause != nil; cause = errors.Unwrap(err2) {
		err2 = cause
	}
	message := err2.Error()
	previous := -1
	for _, key := range []string{"cpudevicemode", "groupby", "reservedcpus"} {
		index := strings.Index(message, `"`+key+`"`)
		require.Greater(t, index, previous, "all keys must appear in sorted order")
		previous = index
	}
}

// TestResolve_PartialFile: fields absent from the file retain their default values.
func TestResolve_PartialFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
reservedCPUs: "4-7"
`)

	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.NoError(t, err)
	assert.Equal(t, "4-7", result.ReservedCPUs)
	assert.Equal(t, ":8080", result.BindAddress)
	assert.Equal(t, device.CPU_DEVICE_MODE_GROUPED, result.CPUDeviceMode)
	assert.Equal(t, device.GROUP_BY_NUMA_NODE, result.GroupBy)
}

// TestResolve_FileWithoutAPIVersion: omitting apiVersion is accepted.
func TestResolve_FileWithoutAPIVersion(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
reservedCPUs: "5-6"
`)

	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.NoError(t, err)
	assert.Equal(t, "5-6", result.ReservedCPUs)
}

// TestResolve_UnknownAPIVersionIsError: an unrecognised apiVersion is rejected.
func TestResolve_UnknownAPIVersionIsError(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v99
reservedCPUs: "0-3"
`)

	_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported apiVersion")
	assert.Contains(t, err.Error(), "v99")
}

// TestResolve_MissingFileIsError: a non-existent file path returns an error.
func TestResolve_MissingFileIsError(t *testing.T) {
	_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile("/does/not/exist/config.yaml"),
	})

	require.Error(t, err)
}

// TestResolve_IgnoresUnrelatedFlags: flags that are not Config fields (such as
// --config and the klog --v flag that share the process FlagSet) must not
// produce any error log, while a mapped flag set on the command line still
// overrides the file. Regression guard for the spurious "flag not found in
// flagToJSONKey" errors that used to fire on every startup with a config file.
func TestResolve_IgnoresUnrelatedFlags(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
reservedCPUs: "0-3"
hostnameOverride: from-file
`)

	cliopts := driverconfig.DefaultCLIOptions()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	// Mirror the real command line: --config and klog's --v live on the same
	// FlagSet but are not Config fields.
	fs.String("config", "", "path to the config file")
	fs.Int("v", 0, "log verbosity")
	cliopts.AddFlags(fs)
	require.NoError(t, fs.Parse([]string{
		"--config=" + cfgFile,
		"--v=4",
		"--kubeconfig=/from-cli",
	}))

	var logs strings.Builder
	logger := funcr.New(func(prefix, args string) {
		logs.WriteString(prefix + " " + args + "\n")
	}, funcr.Options{Verbosity: 10})

	got, err := driverconfig.Resolve(logger, []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
		driverconfig.FromFlags(cliopts),
	})

	require.NoError(t, err)
	assert.NotContains(t, logs.String(), "flag not found",
		"unrelated flags must not produce an error log")
	// The mapped flag set on the command line still wins over the file value.
	assert.Equal(t, "/from-cli", got.Kubeconfig)
	// A file-only field is still applied, so the assertions above cannot pass
	// without the file actually being read.
	assert.Equal(t, "from-file", got.HostnameOverride)
}

// TestResolve_EmptyFilePath: an empty file path is a no-op.
func TestResolve_EmptyFilePath(t *testing.T) {
	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(""),
	})

	require.NoError(t, err)
	assert.Equal(t, driverconfig.Default(), result)
}

func TestFromV1Alpha1Overlay(t *testing.T) {
	cfg := driverconfig.Default()
	cfg.Kubeconfig = "/prior-config"
	cfg.PublishNodeAllocatableResourceMapping = true
	cfg.KubeletRootDir = "/custom/kubelet"

	want := cfg
	driverconfig.FromV1Alpha1(v1alpha1.Config{}, &cfg)
	assert.Equal(t, want, cfg, "omitted fields must leave earlier values intact")

	want.Kubeconfig = ""
	want.PublishNodeAllocatableResourceMapping = false
	driverconfig.FromV1Alpha1(v1alpha1.Config{
		Kubeconfig:                            new(""),
		PublishNodeAllocatableResourceMapping: new(false),
	}, &cfg)
	assert.Equal(t, want, cfg, "explicit zero values must override earlier values")
}

// TestDefault pins the built-in default values.
func TestDefault(t *testing.T) {
	d := driverconfig.Default()

	assert.Equal(t, ":8080", d.BindAddress)
	assert.Equal(t, device.CPU_DEVICE_MODE_GROUPED, d.CPUDeviceMode)
	assert.Equal(t, device.GROUP_BY_NUMA_NODE, d.GroupBy)
	// The kubelet root defaults to the standard location, so behavior is
	// unchanged unless the kubelet --root-dir is relocated.
	assert.Equal(t, driverconfig.DefaultKubeletRootDir, d.KubeletRootDir)
	// Fields with no built-in default must be zero/empty.
	assert.Empty(t, d.Kubeconfig)
	assert.Empty(t, d.HostnameOverride)
	assert.Empty(t, d.ReservedCPUs)
	assert.False(t, d.ExposePCIeRoots)
	assert.False(t, d.PublishNodeAllocatableResourceMapping)
}

// TestResolve_InvalidCPUDeviceModeIsError: an invalid cpuDeviceMode in the file is rejected.
func TestResolve_InvalidCPUDeviceModeIsError(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
cpuDeviceMode: garbage
`)

	_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid cpuDeviceMode")
	assert.Contains(t, err.Error(), "garbage")
}

// TestResolve_InvalidGroupByIsError: an invalid groupBy in the file is rejected.
func TestResolve_InvalidGroupByIsError(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
groupBy: garbage
`)

	_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid groupBy")
	assert.Contains(t, err.Error(), "garbage")
}

// TestResolve_ExcludedFieldInFileIsError: excluded and removed fields aren't
// configurable via the config file.
func TestResolve_ExcludedFieldInFileIsError(t *testing.T) {
	for _, tc := range []struct {
		field    string
		content  string
		wantErrs []string
	}{
		{field: "bindAddress", content: "bindAddress: \":9090\"", wantErrs: []string{"not configurable via the config file"}},
		{field: "exposePCIeRoots", content: "exposePCIeRoots: true", wantErrs: []string{"not configurable via the config file"}},
		{field: "showMetrics", content: "showMetrics: true", wantErrs: []string{"unknown field"}},
		{
			field:   "kubeletRootDir",
			content: "kubeletRootDir: /mnt/fast/k8s/kubelet",
			// The error must name both routes or a Helm user will not know
			// kubeletRootDir is what the chart actually maps the flag to.
			wantErrs: []string{
				"not configurable via the config file",
				"use the chart's top-level kubeletRootDir value",
				"--kubelet-root-dir when running the binary directly",
			},
		},
	} {
		t.Run(tc.field, func(t *testing.T) {
			dir := t.TempDir()
			cfgFile := writeFile(t, dir, "config.yaml", "apiVersion: v1alpha1\n"+tc.content+"\n")

			_, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
				driverconfig.FromFile(cfgFile),
			})

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.field)
			for _, want := range tc.wantErrs {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

// The chart passes the kubelet root as a flag and renders its hostPath mounts
// from the same value, so a root that cannot be used has to fail here.
func TestResolve_KubeletRootDirFromFlag(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []string
		wantRoot string
		wantErr  string
	}{
		{
			name:     "absolute",
			args:     []string{"--kubelet-root-dir=/mnt/fast/k8s/kubelet"},
			wantRoot: "/mnt/fast/k8s/kubelet",
		},
		{
			// Cleaned in the config layer so the logged value matches the paths
			// the driver and the chart derive from it.
			name:     "non-canonical is cleaned",
			args:     []string{"--kubelet-root-dir=/mnt/a/../kubelet//"},
			wantRoot: "/mnt/kubelet",
		},
		{
			name:    "relative",
			args:    []string{"--kubelet-root-dir=relative/kubelet"},
			wantErr: "must be an absolute path",
		},
		{
			name:    "empty",
			args:    []string{"--kubelet-root-dir="},
			wantErr: "must not be empty",
		},
		{
			// flag takes the last value, so a chart appending an empty override
			// would otherwise undo an earlier root.
			name:    "emptied after being set",
			args:    []string{"--kubelet-root-dir=/mnt/fast/k8s/kubelet", "--kubelet-root-dir="},
			wantErr: "must not be empty",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cliopts := driverconfig.DefaultCLIOptions()
			newFlagSet(t, &cliopts, tc.args)

			result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
				driverconfig.FromFlags(cliopts),
			})

			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantRoot, result.KubeletRootDir)
		})
	}
}

func TestFromFlags_OnlyExplicitValuesOverride(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
	}{
		{name: "unset"},
		{name: "explicit zero values", args: []string{"--kubeconfig=", "--expose-pcie-roots=false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cliopts := driverconfig.DefaultCLIOptions()
			newFlagSet(t, &cliopts, tc.args)
			cfg := driverconfig.Default()
			cfg.Kubeconfig = "/from-file"
			cfg.ExposePCIeRoots = true
			cfg.BindAddress = ":9090"
			cfg.KubeletRootDir = "/custom/kubelet"

			require.NoError(t, driverconfig.FromFlags(cliopts).Apply(testr.New(t), &cfg))
			if len(tc.args) == 0 {
				assert.Equal(t, "/from-file", cfg.Kubeconfig)
				assert.True(t, cfg.ExposePCIeRoots)
			} else {
				assert.Empty(t, cfg.Kubeconfig)
				assert.False(t, cfg.ExposePCIeRoots)
			}
			assert.Equal(t, ":9090", cfg.BindAddress)
			assert.Equal(t, "/custom/kubelet", cfg.KubeletRootDir)
		})
	}
}

func TestCLIOptions_TracksSuccessfulParsing(t *testing.T) {
	cliopts := driverconfig.DefaultCLIOptions()
	fs := newFlagSet(t, &cliopts, []string{"--expose-pcie-roots", "--kubeconfig=/first", "--kubeconfig="})
	assert.True(t, cliopts.ExposePCIeRoots.Set)
	assert.True(t, cliopts.ExposePCIeRoots.Value)
	assert.True(t, cliopts.Kubeconfig.Set)
	assert.Empty(t, cliopts.Kubeconfig.Value)
	assert.False(t, cliopts.BindAddress.Set)
	assert.False(t, cliopts.KubeletRootDir.Set)
	assert.Equal(t, driverconfig.DefaultBindAddress, cliopts.BindAddress.Value)
	assert.Equal(t, driverconfig.DefaultKubeletRootDir, cliopts.KubeletRootDir.Value)

	var help strings.Builder
	fs.SetOutput(&help)
	fs.PrintDefaults()
	assert.NotContains(t, help.String(), "panic")
	assert.Contains(t, help.String(), driverconfig.DefaultBindAddress)
	assert.Contains(t, help.String(), driverconfig.DefaultKubeletRootDir)

	cliopts = driverconfig.DefaultCLIOptions()
	fs = flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(&strings.Builder{})
	cliopts.AddFlags(fs)
	require.Error(t, fs.Parse([]string{"--expose-pcie-roots=invalid"}))
	assert.False(t, cliopts.ExposePCIeRoots.Set)
}

// TestFromFlags_BoolAndStringValues checks typed boolean and string overrides.
func TestFromFlags_BoolAndStringValues(t *testing.T) {
	for _, value := range []string{"true", "false"} {
		t.Run(value, func(t *testing.T) {
			cliopts := driverconfig.DefaultCLIOptions()
			newFlagSet(t, &cliopts, []string{"--expose-pcie-roots=" + value, "--kubeconfig=/from-cli"})
			result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{driverconfig.FromFlags(cliopts)})
			require.NoError(t, err)
			assert.Equal(t, value == "true", result.ExposePCIeRoots)
			assert.Equal(t, "/from-cli", result.Kubeconfig)
		})
	}
}

func TestResolve_KubeletRootDirWithAConfigFile(t *testing.T) {
	for _, tc := range []struct {
		name             string
		content          string
		args             []string
		wantErrs         []string
		wantRoot         string
		wantReservedCPUs string
	}{
		{
			// encoding/json matches a field without regard to case, so before the
			// canonical-key pass a differently spelled key walked past the
			// exclusion and replaced a root given on the command line, which is
			// issue #231 reached through a config file.
			name:     "a differently spelled key cannot override the flag",
			content:  "kubeletrootdir: /wrong/root",
			args:     []string{"--kubelet-root-dir=/correct/root"},
			wantErrs: []string{"kubeletrootdir", "not configurable via the config file"},
		},
		{
			// The control. Refusing every file that mentions anything would close
			// the case above without the pass that closes it.
			name:             "an unrelated file leaves the flag alone",
			content:          `reservedCPUs: "0-3"`,
			args:             []string{"--kubelet-root-dir=/correct/root"},
			wantRoot:         "/correct/root",
			wantReservedCPUs: "0-3",
		},
		{
			// The file is an input, not the source: it is valid and is not allowed
			// to carry this field, so blaming its contents would send the reader
			// looking for a key that cannot be there.
			name:     "a bad flag names the file as an input",
			content:  `reservedCPUs: "0-1"`,
			args:     []string{"--kubelet-root-dir=relative/kubelet"},
			wantErrs: []string{"must be an absolute path"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			cfgFile := writeFile(t, dir, "config.yaml", "apiVersion: v1alpha1\n"+tc.content+"\n")

			cliopts := driverconfig.DefaultCLIOptions()
			newFlagSet(t, &cliopts, tc.args)

			result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
				driverconfig.FromFile(cfgFile),
				driverconfig.FromFlags(cliopts),
			})

			if len(tc.wantErrs) > 0 {
				require.Error(t, err)
				for _, want := range tc.wantErrs {
					assert.Contains(t, err.Error(), want)
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantRoot, result.KubeletRootDir)
			assert.Equal(t, tc.wantReservedCPUs, result.ReservedCPUs)
		})
	}
}

// TestResolve_BoolFlagWinsOverFile: a bool CLI flag correctly overrides via the JSON round-trip.
func TestResolve_BoolFlagWinsOverFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := writeFile(t, dir, "config.yaml", `
apiVersion: v1alpha1
reservedCPUs: "0-3"
`)

	cliopts := driverconfig.DefaultCLIOptions()
	newFlagSet(t, &cliopts, []string{
		"--expose-pcie-roots=true",
	})

	result, err := driverconfig.Resolve(testr.New(t), []driverconfig.Source{
		driverconfig.FromFile(cfgFile),
		driverconfig.FromFlags(cliopts),
	})

	require.NoError(t, err)
	assert.True(t, result.ExposePCIeRoots)
	assert.Equal(t, "0-3", result.ReservedCPUs)
}

// TestRemovedFlagsRejected ensures deprecated flags cannot silently be ignored.
func TestRemovedFlagsRejected(t *testing.T) {
	for _, name := range []string{"cpu-device-mode", "group-by", "reserved-cpus", "hostname-override", "sysfs-overlay"} {
		t.Run(name, func(t *testing.T) {
			cliopts := driverconfig.DefaultCLIOptions()
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			cliopts.AddFlags(fs)
			assert.Nil(t, fs.Lookup(name))
			require.Error(t, fs.Parse([]string{"--" + name + "=value"}))
		})
	}
}

func TestWarnDeprecatedPCIeFlag(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		warn bool
	}{
		{name: "unset"},
		{name: "unrelated", args: []string{"--bind-address=:9090"}},
		{name: "true", args: []string{"--expose-pcie-roots=true"}, warn: true},
		{name: "false", args: []string{"--expose-pcie-roots=false"}, warn: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cliopts := driverconfig.DefaultCLIOptions()
			fs := newFlagSet(t, &cliopts, tc.args)
			var logs strings.Builder
			logger := funcr.New(func(prefix, args string) { logs.WriteString(prefix + args) }, funcr.Options{})
			driverconfig.WarnDeprecatedFlags(fs, logger)
			if tc.warn {
				assert.Contains(t, logs.String(), "deprecated")
				assert.Contains(t, logs.String(), "expose-pcie-roots")
			} else {
				assert.Empty(t, logs.String())
			}
			fs.VisitAll(func(f *flag.Flag) {
				assert.Equal(t, f.Name == "expose-pcie-roots", strings.Contains(f.Usage, "DEPRECATED:"), f.Name)
			})
		})
	}
}
