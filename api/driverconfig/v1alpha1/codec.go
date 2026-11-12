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

import (
	"encoding/json"
	"errors"
	"fmt"

	strictjson "sigs.k8s.io/json"
	"sigs.k8s.io/yaml"
)

func Encode(cfg *Config) ([]byte, error) {
	return nil, nil
}

func Decode(data []byte) (*Config, error) {
	jsonData, err := yaml.YAMLToJSONStrict(data)
	if err != nil {
		return nil, fmt.Errorf("convert configuration YAML to JSON: %w", err)
	}
	var wire struct {
		Config
		APIVersion json.RawMessage `json:"apiVersion"`
	}
	strictErrors, err := strictjson.UnmarshalStrict(jsonData, &wire)
	if err != nil {
		return nil, fmt.Errorf("decode %s configuration: %w", APIVersion, err)
	}
	if len(strictErrors) != 0 {
		return nil, fmt.Errorf("check %s configuration field names: %w", APIVersion, errors.Join(strictErrors...))
	}
	if wire.APIVersion != nil {
		var version string
		if err := json.Unmarshal(wire.APIVersion, &version); err != nil {
			return nil, fmt.Errorf("apiVersion must be a string such as %q: %w", APIVersion, err)
		}
		if version != APIVersion {
			return nil, fmt.Errorf("unsupported apiVersion %s: use %q with this decoder", wire.APIVersion, APIVersion)
		}
	}
	return &wire.Config, nil
}
