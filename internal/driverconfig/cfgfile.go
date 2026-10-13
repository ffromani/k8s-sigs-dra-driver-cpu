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
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/config"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/config/v1alpha1"
)

type FileSource struct {
	confPath string
}

func FromFile(confPath string) FileSource {
	return FileSource{
		confPath: confPath,
	}
}

func (fs FileSource) Name() string {
	return fs.confPath
}

func (fs FileSource) String() string {
	return fs.confPath
}

func (fs FileSource) Apply(logger logr.Logger, cfg *Config) error {
	if fs.confPath == "" {
		return nil // nothing to do
	}
	data, err := os.ReadFile(fs.confPath)
	if err != nil {
		return fmt.Errorf("loading conf data from %q: %w", fs.confPath, err)
	}
	ver, ok, err := config.DecodeAPIVersion(data)
	if err != nil {
		return fmt.Errorf("checking apiVersion from %q: %w", fs.confPath, err)
	}
	if !ok {
		ver = v1alpha1.APIVersion /// early backward compat
	}

	var src Source
	switch ver {
	case v1alpha1.APIVersion:
		cfg, err := v1alpha1.Decode(data)
		if err != nil {
			return fmt.Errorf("decoding data: %w", err)
		}
		src = V1Alpha1Source{
			Config: cfg,
		}
		logger.V(6).Info("overrides", "stage", src.Name(), "values", cfg.String())
	default:
		return fmt.Errorf("unsupported apiVersion %q, want %q", ver, v1alpha1.APIVersion)
	}
	return src.Apply(logger, cfg)
}
