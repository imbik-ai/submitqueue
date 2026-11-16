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

func TestIssueTrackOverridesApply(t *testing.T) {
	base := entity.IssueTrackConfig{
		RefreshDelayMs:      10,
		RecoveryWaitDelayMs: 11,
	}

	assert.Equal(t, base, (IssueTrackOverrides{}).Apply(base))

	t.Run("RefreshDelayMs", func(t *testing.T) {
		value := int64(25)
		original := base
		override := IssueTrackOverrides{RefreshDelayMs: &value}

		got := override.Apply(base)
		want := base
		want.RefreshDelayMs = value

		assert.Equal(t, want, got)
		assert.Equal(t, original, base)

		value = 0
		assert.Equal(t, int64(25), got.RefreshDelayMs)
		assert.Zero(t, override.Apply(base).RefreshDelayMs)
	})

	t.Run("RecoveryWaitDelayMs", func(t *testing.T) {
		value := int64(25)
		original := base
		override := IssueTrackOverrides{RecoveryWaitDelayMs: &value}

		got := override.Apply(base)
		want := base
		want.RecoveryWaitDelayMs = value

		assert.Equal(t, want, got)
		assert.Equal(t, original, base)

		value = 0
		assert.Equal(t, int64(25), got.RecoveryWaitDelayMs)
		assert.Zero(t, override.Apply(base).RecoveryWaitDelayMs)
	})

}
