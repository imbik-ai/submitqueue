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

type issueTrackSettings struct {
	RefreshDelay *time.Duration `yaml:"refreshDelay"`

	RecoveryWaitDelay *time.Duration `yaml:"recoveryWaitDelay"`
}

func (s issueTrackSettings) overrides() (queueconfig.IssueTrackOverrides, error) {
	refreshDelay, err := durationMilliseconds(s.RefreshDelay)
	if err != nil {
		return queueconfig.IssueTrackOverrides{}, fmt.Errorf("refreshDelay: %w", err)
	}

	recoveryWaitDelay, err := durationMilliseconds(s.RecoveryWaitDelay)
	if err != nil {
		return queueconfig.IssueTrackOverrides{}, fmt.Errorf("recoveryWaitDelay: %w", err)
	}

	return queueconfig.IssueTrackOverrides{
		RefreshDelayMs:      refreshDelay,
		RecoveryWaitDelayMs: recoveryWaitDelay,
	}, nil
}
