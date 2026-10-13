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

package config

import (
	"fmt"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	KeyAPIVersion = "apiVersion"
)

func DecodeAPIVersion(data []byte) (string, bool, error) {
	var fields map[string]any
	if err := yaml.UnmarshalStrict(data, &fields); err != nil {
		return "", false, fmt.Errorf("parsing field data: %w", err)
	}
	for key := range fields {
		if strings.EqualFold(key, KeyAPIVersion) && key != KeyAPIVersion {
			return "", false, fmt.Errorf("field %q is spelled differently from the schema; use %q", key, KeyAPIVersion)
		}
	}
	raw, present := fields[KeyAPIVersion]
	if !present {
		return "", false, nil
	}
	version, ok := raw.(string)
	if !ok {
		return "", true, fmt.Errorf("version must be a string")
	}
	return version, true, nil
}
