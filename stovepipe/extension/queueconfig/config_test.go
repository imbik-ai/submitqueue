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

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/submitqueue/stovepipe/entity"
)

func TestOverridesComposeAllStages(t *testing.T) {
	concurrency := int32(2)
	admission, accepted, running := int64(11), int64(12), int64(13)
	analysis, refresh, recovery := int64(14), int64(15), int64(16)
	overrides := Overrides{
		Process:       ProcessOverrides{MaxConcurrent: &concurrency, GateWaitDelayMs: &admission},
		BuildSignal:   BuildSignalOverrides{AcceptedPollDelayMs: &accepted, RunningPollDelayMs: &running},
		BuildAnalysis: BuildAnalysisOverrides{PollDelayMs: &analysis},
		IssueTrack:    IssueTrackOverrides{RefreshDelayMs: &refresh, RecoveryWaitDelayMs: &recovery},
	}
	var base entity.StageConfig

	got := overrides.Apply(base)
	require.NoError(t, got.Validate())
	assert.Equal(t, entity.StageConfig{}, base)
	assert.Equal(t, concurrency, got.Process.MaxConcurrent)
	assert.Equal(t, admission, got.Process.GateWaitDelayMs)
	assert.Equal(t, accepted, got.BuildSignal.AcceptedPollDelayMs)
	assert.Equal(t, running, got.BuildSignal.RunningPollDelayMs)
	assert.Equal(t, analysis, got.BuildAnalysis.PollDelayMs)
	assert.Equal(t, refresh, got.IssueTrack.RefreshDelayMs)
	assert.Equal(t, recovery, got.IssueTrack.RecoveryWaitDelayMs)

	inherited := (Overrides{}).Apply(got)
	assert.Equal(t, got, inherited)
}
