// Copyright (c) 2025 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package static provides immutable, resolved queue configuration.
package static

import (
	"context"
	"fmt"
	"sort"

	"github.com/uber/submitqueue/stovepipe/entity"
	"github.com/uber/submitqueue/stovepipe/extension/queueconfig"
)

// Store contains configurations for an explicit set of queues.
// Get and List return independent values and are safe for concurrent use.
type Store struct {
	queues map[string]entity.QueueConfig
	names  []string
}

// NewStore applies ordered per-queue overrides to a complete deployment configuration.
// Unknown override names, duplicate queue names and invalid settings are rejected.
func NewStore(
	names []string,
	deployment entity.StageConfig,
	overrides map[string][]queueconfig.Overrides,
) (Store, error) {
	if err := deployment.Validate(); err != nil {
		return Store{}, fmt.Errorf("deployment configuration: %w", err)
	}

	store := Store{queues: make(map[string]entity.QueueConfig, len(names))}

	for _, name := range names {
		if _, exists := store.queues[name]; exists {
			return Store{}, fmt.Errorf("duplicate queue %q", name)
		}

		stages := deployment

		for _, layer := range overrides[name] {
			stages = layer.Apply(stages)
		}

		cfg := entity.QueueConfig{
			Name:   name,
			Stages: stages,
		}

		if err := cfg.Validate(); err != nil {
			return Store{}, err
		}

		store.queues[name] = cfg
		store.names = append(store.names, name)
	}

	for name := range overrides {
		if _, exists := store.queues[name]; !exists {
			return Store{}, fmt.Errorf("override for unknown queue %q: %w", name, queueconfig.ErrNotFound)
		}
	}

	sort.Strings(store.names)

	return store, nil
}

// Get returns ErrNotFound for queues outside the configured set.
func (s Store) Get(_ context.Context, name string) (entity.QueueConfig, error) {
	cfg, ok := s.queues[name]
	if !ok {
		return entity.QueueConfig{}, queueconfig.ErrNotFound
	}

	return cfg, nil
}

// List returns configurations ordered by queue name.
func (s Store) List(context.Context) ([]entity.QueueConfig, error) {
	configs := make([]entity.QueueConfig, 0, len(s.names))

	for _, name := range s.names {
		configs = append(configs, s.queues[name])
	}

	return configs, nil
}

var _ queueconfig.Store = Store{}
