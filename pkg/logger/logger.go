/*
Copyright 2025 The Kubernetes Authors.

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

package logger

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/klog/v2/textlogger"
)

var (
	fallback logr.Logger
)

func init() {
	fallback = textlogger.NewLogger(textlogger.NewConfig())
}

// SetFallback replaces the fallback logger used when no logger is
// found in a context.Context.
// NOT SAFE to be used concurrently: call this once as early as possible
// from main after creating the "real" logger.
func SetFallback(l logr.Logger) {
	fallback = l
}

// FromContext extracts a logr.Logger from ctx. If none is found it
// returns the global fallback logger.
func FromContext(ctx context.Context) logr.Logger {
	logger, err := logr.FromContext(ctx)
	if err != nil {
		return fallback
	}
	return logger
}
