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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uber/submitqueue/stovepipe/extension/queueconfig"
)

func loadTestLayer(input string, names []string) (queueconfig.Store, error) {
	base, err := decodeDocument(bytes.NewReader(baseYAML))
	if err != nil {
		return nil, err
	}

	layer, err := decodeDocument(strings.NewReader(input))
	if err != nil {
		return nil, err
	}

	return newQueueConfigStore(names, base, layer)
}

func TestStageAndQueueOverrides(t *testing.T) {
	store, err := loadTestLayer(`
queueconfig:
  stages:
    process:
      maxConcurrent: 3
      gateWaitDelay: 123ms
    buildSignal:
      acceptedPollDelay: 234ms
      runningPollDelay: 345ms
    buildAnalysis:
      pollDelay: 456ms
    issueTrack:
      refreshDelay: 567ms
      recoveryWaitDelay: 678ms
  queues:
    special:
      stages:
        process:
          maxConcurrent: 4
        buildAnalysis:
          pollDelay: 2s
`, []string{"special", "ordinary"})
	require.NoError(t, err)

	ordinary, err := store.Get(context.Background(), "ordinary")
	require.NoError(t, err)

	special, err := store.Get(context.Background(), "special")
	require.NoError(t, err)

	assert.Equal(t, int32(3), ordinary.Stages.Process.MaxConcurrent)
	assert.Equal(t, int64(123), ordinary.Stages.Process.GateWaitDelayMs)
	assert.Equal(t, int64(234), ordinary.Stages.BuildSignal.AcceptedPollDelayMs)
	assert.Equal(t, int64(345), ordinary.Stages.BuildSignal.RunningPollDelayMs)
	assert.Equal(t, int64(456), ordinary.Stages.BuildAnalysis.PollDelayMs)
	assert.Equal(t, int64(567), ordinary.Stages.IssueTrack.RefreshDelayMs)
	assert.Equal(t, int64(678), ordinary.Stages.IssueTrack.RecoveryWaitDelayMs)

	assert.Equal(t, int32(4), special.Stages.Process.MaxConcurrent)
	assert.Equal(t, ordinary.Stages.Process.GateWaitDelayMs, special.Stages.Process.GateWaitDelayMs)
	assert.Equal(t, int64(2000), special.Stages.BuildAnalysis.PollDelayMs)
	assert.Equal(t, ordinary.Stages.BuildSignal, special.Stages.BuildSignal)
	assert.Equal(t, ordinary.Stages.IssueTrack, special.Stages.IssueTrack)
}

func TestLayeredQueueSettingsAreMergedBeforeValidation(t *testing.T) {
	base, err := decodeDocument(bytes.NewReader(baseYAML))
	require.NoError(t, err)

	first, err := decodeDocument(strings.NewReader(`
queueconfig:
  stages:
    buildSignal:
      runningPollDelay: 0s
  queues:
    special:
      stages:
        process:
          maxConcurrent: 3
        buildAnalysis:
          pollDelay: 0s
`))
	require.NoError(t, err)

	second, err := decodeDocument(strings.NewReader(`
queueconfig:
  stages:
    buildSignal:
      runningPollDelay: 555ms
  queues:
    special:
      stages:
        process:
          gateWaitDelay: 321ms
        buildAnalysis:
          pollDelay: 100ms
`))
	require.NoError(t, err)

	store, err := newQueueConfigStore([]string{"special"}, base, first, second)
	require.NoError(t, err)

	cfg, err := store.Get(context.Background(), "special")
	require.NoError(t, err)

	assert.Equal(t, int32(3), cfg.Stages.Process.MaxConcurrent)
	assert.Equal(t, int64(321), cfg.Stages.Process.GateWaitDelayMs)
	assert.Equal(t, int64(100), cfg.Stages.BuildAnalysis.PollDelayMs)
	assert.Equal(t, int64(555), cfg.Stages.BuildSignal.RunningPollDelayMs)
}

func TestRejectsInvalidConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"unknown section", "queueconfg: {}"},
		{"unknown stage", "queueconfig: {stages: {unknown: {}}}"},
		{"unknown process field", "queueconfig: {stages: {process: {maxConcurent: 2}}}"},
		{"unknown signal field", "queueconfig: {stages: {buildSignal: {pollDelay: 1s}}}"},
		{"unknown analysis field", "queueconfig: {stages: {buildAnalysis: {polDelay: 1s}}}"},
		{"unknown tracking field", "queueconfig: {stages: {issueTrack: {pollDelay: 1s}}}"},
		{"unknown queue", "queueconfig: {queues: {unknown: {}}}"},
		{"duplicate key", "queueconfig: {stages: {process: {maxConcurrent: 1, maxConcurrent: 2}}}"},
		{"zero concurrency", "queueconfig: {stages: {process: {maxConcurrent: 0}}}"},
		{"zero wait", "queueconfig: {stages: {buildAnalysis: {pollDelay: 0s}}}"},
		{"negative wait", "queueconfig: {stages: {issueTrack: {refreshDelay: -1s}}}"},
		{"invalid duration", "queueconfig: {stages: {process: {gateWaitDelay: soon}}}"},
		{"submillisecond", "queueconfig: {stages: {buildSignal: {acceptedPollDelay: 1us}}}"},
		{"fractional millisecond", "queueconfig: {stages: {issueTrack: {recoveryWaitDelay: 1500us}}}"},
		{"invalid queue override", "queueconfig: {queues: {known: {stages: {process: {gateWaitDelay: 0s}}}}}"},
		{"second document", "{}\n---\n{}"},
		{"malformed second document", "{}\n---\n["},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadTestLayer(tt.input, []string{"known"})
			require.Error(t, err)
		})
	}
}

func TestLoad(t *testing.T) {
	ctx := context.Background()

	t.Run("embedded baseline and explicit queue registry", func(t *testing.T) {
		store, err := Load("", []string{"known"})
		require.NoError(t, err)

		cfg, err := store.Get(ctx, "known")
		require.NoError(t, err)
		require.NoError(t, cfg.Validate())

		_, err = store.Get(ctx, "unknown")
		require.ErrorIs(t, err, queueconfig.ErrNotFound)
	})

	t.Run("optional file overlays rather than replaces baseline", func(t *testing.T) {
		baseline, err := Load("", []string{"known"})
		require.NoError(t, err)

		want, err := baseline.Get(ctx, "known")
		require.NoError(t, err)
		want.Stages.Process.GateWaitDelayMs = 321

		path := filepath.Join(t.TempDir(), "stovepipe.yaml")
		input := "queueconfig: {stages: {process: {gateWaitDelay: 321ms}}}"
		require.NoError(t, os.WriteFile(path, []byte(input), 0600))

		store, err := Load(path, []string{"known"})
		require.NoError(t, err)

		got, err := store.Get(ctx, "known")
		require.NoError(t, err)
		assert.Equal(t, want, got)
	})

	t.Run("missing explicit file fails", func(t *testing.T) {
		_, err := Load(filepath.Join(t.TempDir(), "missing"), []string{"known"})
		require.ErrorIs(t, err, os.ErrNotExist)
	})

	t.Run("incomplete baseline is not silently defaulted", func(t *testing.T) {
		partial, err := decodeDocument(strings.NewReader("queueconfig: {stages: {process: {maxConcurrent: 1}}}"))
		require.NoError(t, err)

		_, err = newQueueConfigStore(nil, partial)
		require.Error(t, err)
	})
}
