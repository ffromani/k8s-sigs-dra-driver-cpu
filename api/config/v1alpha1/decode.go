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
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/kubernetes-sigs/dra-driver-cpu/api/config"
	"sigs.k8s.io/json"
	"sigs.k8s.io/yaml"
)

func Decode(data []byte) (Config, error) {
	// since this version accepts missing apiversion and treat it
	// as v1alpha1, we need extra care. Following versions will
	// just require the apiversion to be present, so their Decode
	// function can make assumptions.
	version, ok, err := config.DecodeAPIVersion(data)
	if err != nil {
		return Config{}, err
	}
	if ok && version != APIVersion {
		return Config{}, fmt.Errorf("version must be %q if specified", APIVersion)
	}

	// make sure the internal json conversion doesn't fold case. So we do ourselves.
	jsonData, err := yaml.YAMLToJSONStrict(data)
	if err != nil {
		return Config{}, err
	}
	// now the real decode
	cfg := Config{}
	strictErrs, err := json.UnmarshalStrict(jsonData, &cfg)
	if err != nil {
		return Config{}, improveError(fmt.Errorf("parsing data: %w", err), data)
	}
	if len(strictErrs) > 0 {
		return Config{}, improveError(errors.Join(strictErrs...), data)
	}
	return cfg, nil
}

// excludedFieldHints explain settings that are not part of this file format.
var excludedFieldHints = map[string]string{
	"bindAddress":     "use helm chart's healthzPort instead",
	"exposePCIeRoots": "use args.exposePCIeRoots instead",
	"kubeletRootDir":  "use the chart's top-level kubeletRootDir value, or --kubelet-root-dir when running the binary directly",
}

func improveError(err error, data []byte) error {
	// Improve diagnostics for fields already rejected by the public schema.
	var fields map[string]any
	if yaml.UnmarshalStrict(data, &fields) == nil {
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			for name, hint := range excludedFieldHints {
				if strings.EqualFold(key, name) {
					return fmt.Errorf("field %q is not configurable via the config file; %s", key, hint)
				}
			}
		}
	}
	return err
}
