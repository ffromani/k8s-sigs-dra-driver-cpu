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
	"path/filepath"

	"github.com/go-logr/logr"
)

type Source interface {
	fmt.Stringer
	Name() string
	Apply(logger logr.Logger, cfg *Config) error
}

func Resolve(logger logr.Logger, sources []Source) (Config, error) {
	cfg := Default()
	logger.WithValues("stage", "default").V(6).Info("config", cfg.LogValues()...)

	for _, src := range sources {
		if err := src.Apply(logger, &cfg); err != nil {
			return Config{}, fmt.Errorf("cannot apply %s: %w", src.Name(), err)
		}
		logger.WithValues("applied", src.Name(), "source", src.String()).V(6).Info("config", cfg.LogValues()...)
	}
	if cfg.KubeletRootDir != "" {
		cfg.KubeletRootDir = filepath.Clean(cfg.KubeletRootDir)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("cannot validate config: %w", err)
	}
	logger.WithValues("stage", "validated").V(6).Info("config", cfg.LogValues()...)
	return cfg, nil
}
