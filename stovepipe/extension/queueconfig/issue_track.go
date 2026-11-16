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

// IssueTrackOverrides contains optional configuration for the issue-tracking stage.
// Nil fields inherit their previous value; explicit values are retained for validation.
type IssueTrackOverrides struct {
	// RefreshDelayMs overrides the interval between refreshes, in milliseconds.
	RefreshDelayMs *int64

	// RecoveryWaitDelayMs overrides the wait during recovery, in milliseconds.
	RecoveryWaitDelayMs *int64
}

// Apply returns a copy of base with explicitly supplied settings replaced.
func (o IssueTrackOverrides) Apply(base entity.IssueTrackConfig) entity.IssueTrackConfig {
	if o.RefreshDelayMs != nil {
		base.RefreshDelayMs = *o.RefreshDelayMs
	}

	if o.RecoveryWaitDelayMs != nil {
		base.RecoveryWaitDelayMs = *o.RecoveryWaitDelayMs
	}

	return base
}
