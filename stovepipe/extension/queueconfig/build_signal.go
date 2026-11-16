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

package queueconfig

import "github.com/uber/submitqueue/stovepipe/entity"

// BuildSignalOverrides contains optional configuration for the build-signal stage.
// Nil fields inherit their previous value; explicit values are retained for validation.
type BuildSignalOverrides struct {
	// AcceptedPollDelayMs overrides the polling interval for accepted builds, in milliseconds.
	AcceptedPollDelayMs *int64

	// RunningPollDelayMs overrides the polling interval for running builds, in milliseconds.
	RunningPollDelayMs *int64
}

// Apply returns a copy of base with explicitly supplied settings replaced.
func (o BuildSignalOverrides) Apply(base entity.BuildSignalConfig) entity.BuildSignalConfig {
	if o.AcceptedPollDelayMs != nil {
		base.AcceptedPollDelayMs = *o.AcceptedPollDelayMs
	}

	if o.RunningPollDelayMs != nil {
		base.RunningPollDelayMs = *o.RunningPollDelayMs
	}

	return base
}
