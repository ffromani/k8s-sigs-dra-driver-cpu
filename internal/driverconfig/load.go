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
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig/v1alpha1"
	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/cpuallocator"
	"github.com/kubernetes-sigs/dra-driver-cpu/pkg/device"
)

type Source interface {
	Name() string
	Apply(logger logr.Logger, cfg *driverconfig.Config) error
}

func Resolve(logger logr.Logger, sources []Source) (driverconfig.Config, error) {
	cfg := driverconfig.Config{
		BindAddress:    driverconfig.DefaultBindAddress,
		KubeletRootDir: driverconfig.DefaultKubeletRootDir,
	}
	defaults := v1alpha1.Default()
	if err := v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(&defaults, &cfg); err != nil {
		return driverconfig.Config{}, fmt.Errorf("cannot apply %s defaults: %w", v1alpha1.APIVersion, err)
	}
	logger.WithValues("stage", "default").V(6).Info("config", cfg.LogValues()...)

	for _, src := range sources {
		if err := src.Apply(logger, &cfg); err != nil {
			return driverconfig.Config{}, fmt.Errorf("cannot apply %s: %w", src.Name(), err)
		}
		logger.WithValues("applied", src.Name()).V(6).Info("config", cfg.LogValues()...)
	}
	if cfg.KubeletRootDir != "" {
		cfg.KubeletRootDir = filepath.Clean(cfg.KubeletRootDir)
	}
	if err := v1alpha1.ValidateKubeletRootDir(cfg.KubeletRootDir); err != nil {
		return driverconfig.Config{}, fmt.Errorf("cannot validate config: %w", err)
	}
	if cfg.CPUDeviceMode == device.CPU_DEVICE_MODE_GROUPED && cfg.GroupBy == device.GROUP_BY_MACHINE && cfg.Allocator != cpuallocator.ModeExternal {
		return driverconfig.Config{}, fmt.Errorf("invalid allocator %q with groupBy %q: must be %q", cfg.Allocator, cfg.GroupBy, cpuallocator.ModeExternal)
	}
	logger.WithValues("stage", "validated").V(6).Info("config", cfg.LogValues()...)
	return cfg, nil
}

// applyMap applies only the keys present in m to cfg; absent keys are
// untouched (encoding/json.Unmarshal semantics). Unknown keys are rejected
// to catch typos early rather than silently ignoring them.
func applyMap(cfg *driverconfig.Config, m map[string]any) error {
	data, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("marshaling config map: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(cfg); err != nil {
		return fmt.Errorf("applying config map: %w", err)
	}
	return nil
}
