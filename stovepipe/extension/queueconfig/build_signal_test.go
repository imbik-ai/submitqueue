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
	"github.com/uber/submitqueue/stovepipe/entity"
)

func TestBuildSignalOverridesApply(t *testing.T) {
	base := entity.BuildSignalConfig{
		AcceptedPollDelayMs: 10,
		RunningPollDelayMs:  11,
	}

	assert.Equal(t, base, (BuildSignalOverrides{}).Apply(base))

	t.Run("AcceptedPollDelayMs", func(t *testing.T) {
		value := int64(25)
		original := base
		override := BuildSignalOverrides{AcceptedPollDelayMs: &value}

		got := override.Apply(base)
		want := base
		want.AcceptedPollDelayMs = value

		assert.Equal(t, want, got)
		assert.Equal(t, original, base)

		value = 0
		assert.Equal(t, int64(25), got.AcceptedPollDelayMs)
		assert.Zero(t, override.Apply(base).AcceptedPollDelayMs)
	})

	t.Run("RunningPollDelayMs", func(t *testing.T) {
		value := int64(25)
		original := base
		override := BuildSignalOverrides{RunningPollDelayMs: &value}

		got := override.Apply(base)
		want := base
		want.RunningPollDelayMs = value

		assert.Equal(t, want, got)
		assert.Equal(t, original, base)

		value = 0
		assert.Equal(t, int64(25), got.RunningPollDelayMs)
		assert.Zero(t, override.Apply(base).RunningPollDelayMs)
	})

}
