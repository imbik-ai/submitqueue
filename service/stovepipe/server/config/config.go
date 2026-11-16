// Copyright (c) 2026 Uber Technologies, Inc.
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

// Package config loads the reference service's layered startup configuration.
package config

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"

	"github.com/uber/submitqueue/stovepipe/entity"
	"github.com/uber/submitqueue/stovepipe/extension/queueconfig"
	"github.com/uber/submitqueue/stovepipe/extension/queueconfig/static"
	"gopkg.in/yaml.v3"
)

//go:embed base.yaml
var baseYAML []byte

type document struct {
	QueueConfig queueSection `yaml:"queueconfig"`
}

type queueSection struct {
	queueSettings `yaml:",inline"`

	Queues map[string]queueSettings `yaml:"queues"`
}

type queueSettings struct {
	Stages stageSettings `yaml:"stages"`
}

// Load layers an optional file over the embedded service baseline.
// Queue-specific settings override deployment settings. Updates require a restart.
func Load(path string, names []string) (queueconfig.Store, error) {
	base, err := decodeDocument(bytes.NewReader(baseYAML))
	if err != nil {
		return nil, fmt.Errorf("decode service baseline: %w", err)
	}

	layers := []document{base}

	if path != "" {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open configuration %q: %w", path, err)
		}
		defer file.Close()

		layer, err := decodeDocument(file)
		if err != nil {
			return nil, fmt.Errorf("decode configuration %q: %w", path, err)
		}

		layers = append(layers, layer)
	}

	return newQueueConfigStore(names, layers...)
}

func decodeDocument(reader io.Reader) (document, error) {
	decoder := yaml.NewDecoder(reader)
	decoder.KnownFields(true)

	var cfg document
	if err := decoder.Decode(&cfg); err != nil {
		return document{}, err
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return document{}, err
		}

		return document{}, fmt.Errorf("configuration must contain exactly one YAML document")
	}

	return cfg, nil
}

func newQueueConfigStore(names []string, layers ...document) (queueconfig.Store, error) {
	var deployment entity.StageConfig
	overrides := make(map[string][]queueconfig.Overrides)

	for _, layer := range layers {
		common, err := layer.QueueConfig.Stages.overrides()
		if err != nil {
			return nil, fmt.Errorf("deployment configuration: %w", err)
		}

		deployment = common.Apply(deployment)

		for name, settings := range layer.QueueConfig.Queues {
			override, err := settings.Stages.overrides()
			if err != nil {
				return nil, fmt.Errorf("queue %q configuration: %w", name, err)
			}

			overrides[name] = append(overrides[name], override)
		}
	}

	return static.NewStore(names, deployment, overrides)
}
