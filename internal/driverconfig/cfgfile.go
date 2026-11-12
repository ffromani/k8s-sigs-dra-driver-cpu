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
	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig"
	"github.com/kubernetes-sigs/dra-driver-cpu/api/driverconfig/v1alpha1"
	"sigs.k8s.io/yaml"
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

func (fs FileSource) Apply(logger logr.Logger, cfg *driverconfig.Config) error {
	if fs.confPath == "" {
		return nil // nothing to do
	}
	data, version, err := loadFile(fs.confPath)
	if err != nil {
		return err
	}
	switch version {
	case v1alpha1.APIVersion:
		loaded, err := v1alpha1.Decode(data)
		if err != nil {
			return fmt.Errorf("decode %s configuration in %q: %w", version, fs.confPath, err)
		}
		logger.V(6).Info("overrides", "stage", fs.Name(), "values", loaded)
		if err := v1alpha1.Convert_v1alpha1_Config_To_driverconfig_Config(loaded, cfg); err != nil {
			return fmt.Errorf("convert %s configuration in %q: %w", version, fs.confPath, err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported apiVersion %q in %q: supported versions: %s; migrate the configuration to a supported version or use a compatible driver", version, fs.confPath, v1alpha1.APIVersion)
	}
}

// loadFile inspects metadata without choosing a schema by trial decoding.
func loadFile(path string) ([]byte, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("reading %s: %w", path, err)
	}
	var metadata map[string]any
	if err := yaml.UnmarshalStrict(data, &metadata); err != nil {
		return nil, "", fmt.Errorf("parsing %s: %w", path, err)
	}
	version := v1alpha1.APIVersion
	if raw, ok := metadata["apiVersion"]; ok {
		version, ok = raw.(string)
		if !ok {
			return nil, "", fmt.Errorf("apiVersion in %q must be a string; set apiVersion: %s", path, v1alpha1.APIVersion)
		}
	}
	return data, version, nil
}
