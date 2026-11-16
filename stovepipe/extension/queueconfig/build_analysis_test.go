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

func TestBuildAnalysisOverridesApply(t *testing.T) {
	base := entity.BuildAnalysisConfig{
		PollDelayMs: 10,
	}

	assert.Equal(t, base, (BuildAnalysisOverrides{}).Apply(base))

	t.Run("PollDelayMs", func(t *testing.T) {
		value := int64(25)
		original := base
		override := BuildAnalysisOverrides{PollDelayMs: &value}

		got := override.Apply(base)
		want := base
		want.PollDelayMs = value

		assert.Equal(t, want, got)
		assert.Equal(t, original, base)

		value = 0
		assert.Equal(t, int64(25), got.PollDelayMs)
		assert.Zero(t, override.Apply(base).PollDelayMs)
	})

}
