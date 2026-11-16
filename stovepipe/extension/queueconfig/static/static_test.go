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

package static

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/submitqueue/stovepipe/entity"
	"github.com/uber/submitqueue/stovepipe/extension/queueconfig"
)

func testStages() entity.StageConfig {
	return entity.StageConfig{
		Process:       entity.ProcessConfig{MaxConcurrent: 2, GateWaitDelayMs: 11},
		BuildSignal:   entity.BuildSignalConfig{AcceptedPollDelayMs: 12, RunningPollDelayMs: 13},
		BuildAnalysis: entity.BuildAnalysisConfig{PollDelayMs: 14},
		IssueTrack:    entity.IssueTrackConfig{RefreshDelayMs: 15, RecoveryWaitDelayMs: 16},
	}
}

func TestStoreInheritanceAndIsolation(t *testing.T) {
	ctx := context.Background()
	deployment := testStages()
	poll := int64(123)
	overrides := map[string][]queueconfig.Overrides{
		"special": {{BuildAnalysis: queueconfig.BuildAnalysisOverrides{PollDelayMs: &poll}}},
	}
	names := []string{"special", "ordinary"}

	store, err := NewStore(names, deployment, overrides)
	require.NoError(t, err)

	deployment.Process.MaxConcurrent = 99
	poll = 0
	names[0] = "mutated"
	delete(overrides, "special")

	special, err := store.Get(ctx, "special")
	require.NoError(t, err)

	ordinary, err := store.Get(ctx, "ordinary")
	require.NoError(t, err)

	assert.Equal(t, int64(123), special.Stages.BuildAnalysis.PollDelayMs)
	assert.Equal(t, testStages(), ordinary.Stages)
	assert.Equal(t, ordinary.Stages.Process, special.Stages.Process)
	assert.Equal(t, ordinary.Stages.BuildSignal, special.Stages.BuildSignal)
	assert.Equal(t, ordinary.Stages.IssueTrack, special.Stages.IssueTrack)

	configs, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, configs, 2)
	assert.Equal(t, []string{"ordinary", "special"}, []string{configs[0].Name, configs[1].Name})

	configs[1].Stages.BuildAnalysis.PollDelayMs = 0
	again, err := store.Get(ctx, "special")
	require.NoError(t, err)
	assert.Equal(t, special, again)

	for _, name := range []string{"", "unknown"} {
		_, err := store.Get(ctx, name)
		require.ErrorIs(t, err, queueconfig.ErrNotFound)
	}
}

func TestStoreValidatesAfterAllQueueLayers(t *testing.T) {
	zero, final := int64(0), int64(250)
	overrides := map[string][]queueconfig.Overrides{
		"known": {
			{BuildAnalysis: queueconfig.BuildAnalysisOverrides{PollDelayMs: &zero}},
			{BuildAnalysis: queueconfig.BuildAnalysisOverrides{PollDelayMs: &final}},
		},
	}

	store, err := NewStore([]string{"known"}, testStages(), overrides)
	require.NoError(t, err)

	cfg, err := store.Get(context.Background(), "known")
	require.NoError(t, err)
	assert.Equal(t, final, cfg.Stages.BuildAnalysis.PollDelayMs)

	overrides["known"] = overrides["known"][:1]
	_, err = NewStore([]string{"known"}, testStages(), overrides)
	require.Error(t, err)
}

func TestStoreRejectsInvalidRegistry(t *testing.T) {
	for _, tt := range []struct {
		name      string
		names     []string
		overrides map[string][]queueconfig.Overrides
	}{
		{name: "empty name", names: []string{""}},
		{name: "duplicate name", names: []string{"a", "a"}},
		{name: "unknown override", names: []string{"a"}, overrides: map[string][]queueconfig.Overrides{"unknown": nil}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewStore(tt.names, testStages(), tt.overrides)
			require.Error(t, err)
		})
	}
}

func TestStoreRequiresValidDeploymentEvenWithoutQueues(t *testing.T) {
	_, err := NewStore(nil, entity.StageConfig{}, nil)
	require.Error(t, err)

	store, err := NewStore(nil, testStages(), nil)
	require.NoError(t, err)

	queues, err := store.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, queues)
}
