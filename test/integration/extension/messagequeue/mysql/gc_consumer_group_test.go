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

package mysql

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	extqueue "github.com/uber/submitqueue/platform/extension/messagequeue"
	queueMySQL "github.com/uber/submitqueue/platform/extension/messagequeue/mysql"
)

// receiveWhileDrainingSignals waits for one delivery while consuming signalCh.
// The subscriber's signal sends block, so a full signal channel would stall
// the worker that is expected to deliver.
func receiveWhileDrainingSignals(t *testing.T, deliveryChan <-chan extqueue.Delivery, signalCh <-chan queueMySQL.HookSignal) extqueue.Delivery {
	t.Helper()
	for {
		select {
		case delivery, ok := <-deliveryChan:
			require.True(t, ok, "delivery channel closed")
			require.NotNil(t, delivery, "received nil delivery")
			return delivery
		case <-signalCh:
		}
	}
}

// TestGCIgnoresConsumerGroupWithoutOffsetRow records the current garbage
// collection behavior for a topic with two consumer groups. GC deletes up to
// the minimum offset_acked over the queue_offsets rows that exist, so a group
// with no row for the partition does not hold messages back: group A acks a
// backlog while group B is not consuming, and GC deletes messages group B has
// never read. The last case is the control, where group B's row exists and
// the backlog survives.
//
// The "lost" cases assert the behavior as it is today. Once GC accounts
// for groups without an offset row, group B must receive the backlog before
// the marker in every case.
func (s *SQLQueueIntegrationSuite) TestGCIgnoresConsumerGroupWithoutOffsetRow() {
	const (
		partition   = "gc-group-part"
		groupA      = "gc-group-a"
		groupB      = "gc-group-b"
		backlogSize = 5
		markerID    = "marker"
		// The subscriber garbage collects every 100 poll ticks; two periods
		// guarantee at least one full GC pass after the last ack.
		pollTicksCoveringGC = 200
		// Group A's GC period is 100 ticks of this interval, about one second.
		groupAPollIntervalMs = 10
		// A running group B re-acquires a partition only on its discovery tick.
		// This interval is longer than the idle grace (2x the 3s test lease
		// duration) and several times group A's GC period, so group A's GC runs
		// between two of group B's discovery ticks.
		slowDiscoveryIntervalMs = 8000
	)

	tests := []struct {
		name string
		// topic is unique per case because the suite shares one database.
		topic string
		// groupBConsumedBefore runs group B through one message before the backlog.
		groupBConsumedBefore bool
		// groupBStopsAfterIdleRelease stops group B only after its idle lease
		// release deleted its offset row; otherwise it stops with the row present.
		groupBStopsAfterIdleRelease bool
		// groupBKeepsRunning leaves group B subscribed for the whole case, with
		// slowDiscoveryIntervalMs, after idle release deleted its offset row.
		groupBKeepsRunning bool
		// wantGroupBReceivesBacklog is whether group B gets the backlog it never read.
		wantGroupBReceivesBacklog bool
	}{
		{
			name:                      "group that never subscribed loses the backlog",
			topic:                     "gc_group_never_subscribed_topic",
			wantGroupBReceivesBacklog: false,
		},
		{
			name:                        "group stopped after idle release deleted its offset row loses the backlog",
			topic:                       "gc_group_idle_released_topic",
			groupBConsumedBefore:        true,
			groupBStopsAfterIdleRelease: true,
			wantGroupBReceivesBacklog:   false,
		},
		{
			name:                      "running group that has not polled again after idle release loses the backlog",
			topic:                     "gc_group_running_idle_released_topic",
			groupBConsumedBefore:      true,
			groupBKeepsRunning:        true,
			wantGroupBReceivesBacklog: false,
		},
		{
			name:                      "group stopped while its offset row exists keeps the backlog",
			topic:                     "gc_group_offset_row_present_topic",
			groupBConsumedBefore:      true,
			wantGroupBReceivesBacklog: true,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			t := s.T()

			signalCh := make(chan queueMySQL.HookSignal, 100)
			queueA, err := queueMySQL.NewQueue(s.testQueueParams(t, func(p *queueMySQL.Params) {
				p.OnSignal = signalCh
			}))
			require.NoError(t, err)
			defer queueA.Close()

			// The warmup delivery stays unacked until group B has consumed it, so
			// its visibility timeout must outlast group B's slow discovery.
			configA := testSubConfig("worker-a", groupA)
			configA.PollIntervalMs = groupAPollIntervalMs
			configA.VisibilityTimeoutMs = 60000
			deliveriesA, err := queueA.Subscriber().Subscribe(s.ctx, tt.topic, configA)
			require.NoError(t, err)

			countMessages := func() int {
				var n int
				require.NoError(t, s.db.QueryRowContext(s.ctx,
					"SELECT COUNT(*) FROM queue_messages WHERE tenant = ? AND topic = ? AND partition_key = ?",
					testTenant, tt.topic, partition).Scan(&n))
				return n
			}
			countOffsetRows := func() int {
				var n int
				require.NoError(t, s.db.QueryRowContext(s.ctx,
					"SELECT COUNT(*) FROM queue_offsets WHERE tenant = ? AND topic = ? AND partition_key = ?",
					testTenant, tt.topic, partition).Scan(&n))
				return n
			}
			groupBAckedOffset := func() (int64, bool) {
				var offset int64
				err := s.db.QueryRowContext(s.ctx,
					"SELECT offset_acked FROM queue_offsets WHERE tenant = ? AND topic = ? AND partition_key = ? AND consumer_group = ?",
					testTenant, tt.topic, partition, groupB).Scan(&offset)
				if errors.Is(err, sql.ErrNoRows) {
					return 0, false
				}
				require.NoError(t, err)
				return offset, true
			}

			var deliveriesB <-chan extqueue.Delivery
			if tt.groupBConsumedBefore {
				configB := testSubConfig("worker-b", groupB)
				if tt.groupBKeepsRunning {
					configB.PartitionDiscoveryIntervalMs = slowDiscoveryIntervalMs
				}
				queueB, err := queueMySQL.NewQueue(s.testQueueParams(t, nil))
				require.NoError(t, err)
				defer queueB.Close()
				deliveriesB, err = queueB.Subscriber().Subscribe(s.ctx, tt.topic, configB)
				require.NoError(t, err)

				require.NoError(t, queueA.Publisher().Publish(s.ctx, tt.topic,
					testMessage("warmup", []byte("w"), partition, nil)))
				// Group A acks last: acking first would let its GC delete the
				// warmup before group B's first poll creates an offset row.
				warmupForGroupA := receiveWhileDrainingSignals(t, deliveriesA, signalCh)
				require.NoError(t, receiveWhileDrainingSignals(t, deliveriesB, signalCh).Ack(s.ctx))
				waitForCondition(t, signalCh, func() bool {
					offset, found := groupBAckedOffset()
					return found && offset > 0
				}, "group B's watermark should advance past the warmup message")
				require.NoError(t, warmupForGroupA.Ack(s.ctx))

				if tt.groupBStopsAfterIdleRelease || tt.groupBKeepsRunning {
					waitForCondition(t, signalCh, func() bool {
						return countOffsetRows() == 0
					}, "both groups should release the drained partition and delete their offset rows")
				}
				if !tt.groupBKeepsRunning {
					require.NoError(t, queueB.Close())
					deliveriesB = nil
					_, groupBRowExists := groupBAckedOffset()
					require.Equal(t, !tt.groupBStopsAfterIdleRelease, groupBRowExists,
						"group B's offset row presence when it stops")
				}
			}

			backlogIDs := make([]string, backlogSize)
			for i := range backlogIDs {
				backlogIDs[i] = fmt.Sprintf("backlog-%d", i)
				require.NoError(t, queueA.Publisher().Publish(s.ctx, tt.topic,
					testMessage(backlogIDs[i], []byte("b"), partition, nil)))
			}
			for _, wantID := range backlogIDs {
				delivery := receiveWhileDrainingSignals(t, deliveriesA, signalCh)
				require.Equal(t, wantID, delivery.Message().ID)
				require.NoError(t, delivery.Ack(s.ctx))
			}

			wantGroupBDeliveries := []string{markerID}
			if tt.wantGroupBReceivesBacklog {
				wantGroupBDeliveries = append(append([]string{}, backlogIDs...), markerID)
				assertNoDelivery(t, deliveriesA, signalCh, queueMySQL.SignalDeliveryCheck, pollTicksCoveringGC)
				require.Equal(t, backlogSize, countMessages(),
					"GC must stop at group B's watermark and keep the backlog")
			} else {
				waitForCondition(t, signalCh, func() bool {
					return countMessages() == 0
				}, "GC deletes the backlog that only group A has acked")
			}

			if deliveriesB == nil {
				queueB, err := queueMySQL.NewQueue(s.testQueueParams(t, nil))
				require.NoError(t, err)
				defer queueB.Close()
				deliveriesB, err = queueB.Subscriber().Subscribe(s.ctx, tt.topic, testSubConfig("worker-b", groupB))
				require.NoError(t, err)
			} else {
				_, groupBRowExists := groupBAckedOffset()
				require.False(t, groupBRowExists,
					"group A's GC ran before the running group B polled the partition again")
			}

			require.NoError(t, queueA.Publisher().Publish(s.ctx, tt.topic,
				testMessage(markerID, []byte("m"), partition, nil)))

			var groupBDeliveries []string
			for len(groupBDeliveries) == 0 || groupBDeliveries[len(groupBDeliveries)-1] != markerID {
				delivery := receiveWhileDrainingSignals(t, deliveriesB, signalCh)
				groupBDeliveries = append(groupBDeliveries, delivery.Message().ID)
				require.NoError(t, delivery.Ack(s.ctx))
			}
			t.Logf("group B received %v", groupBDeliveries)
			require.Equal(t, wantGroupBDeliveries, groupBDeliveries)
		})
	}
}
