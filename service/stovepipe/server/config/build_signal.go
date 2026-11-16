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

package config

import (
	"fmt"
	"time"

	"github.com/uber/submitqueue/stovepipe/extension/queueconfig"
)

type buildSignalSettings struct {
	AcceptedPollDelay *time.Duration `yaml:"acceptedPollDelay"`

	RunningPollDelay *time.Duration `yaml:"runningPollDelay"`
}

func (s buildSignalSettings) overrides() (queueconfig.BuildSignalOverrides, error) {
	acceptedPollDelay, err := durationMilliseconds(s.AcceptedPollDelay)
	if err != nil {
		return queueconfig.BuildSignalOverrides{}, fmt.Errorf("acceptedPollDelay: %w", err)
	}

	runningPollDelay, err := durationMilliseconds(s.RunningPollDelay)
	if err != nil {
		return queueconfig.BuildSignalOverrides{}, fmt.Errorf("runningPollDelay: %w", err)
	}

	return queueconfig.BuildSignalOverrides{
		AcceptedPollDelayMs: acceptedPollDelay,
		RunningPollDelayMs:  runningPollDelay,
	}, nil
}
