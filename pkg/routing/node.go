// Copyright 2023 LiveKit, Inc.
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

package routing

import (
	"runtime"
	"sync"
	"time"

	"github.com/a4to/protocol/conkit"
	"github.com/a4to/protocol/logger"
	"github.com/a4to/protocol/utils"
	"github.com/a4to/protocol/utils/guid"

	"github.com/a4to/conkit-server/pkg/config"
	"github.com/a4to/conkit-server/pkg/telemetry/prometheus"
)

type LocalNode interface {
	Clone() *conkit.Node
	SetNodeID(nodeID conkit.NodeID)
	NodeID() conkit.NodeID
	NodeType() conkit.NodeType
	NodeIP() string
	Region() string
	SetState(state conkit.NodeState)
	SetStats(stats *conkit.NodeStats)
	UpdateNodeStats() bool
	SecondsSinceNodeStatsUpdate() float64
}

type LocalNodeImpl struct {
	lock sync.RWMutex
	node *conkit.Node

	// previous stats for computing averages
	prevStats *conkit.NodeStats
}

func NewLocalNode(conf *config.Config) (*LocalNodeImpl, error) {
	nodeID := guid.New(utils.NodePrefix)
	if conf != nil && conf.RTC.NodeIP == "" {
		return nil, ErrIPNotSet
	}
	l := &LocalNodeImpl{
		node: &conkit.Node{
			Id:      nodeID,
			NumCpus: uint32(runtime.NumCPU()),
			State:   conkit.NodeState_SERVING,
			Stats: &conkit.NodeStats{
				StartedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			},
		},
	}
	if conf != nil {
		l.node.Ip = conf.RTC.NodeIP
		l.node.Region = conf.Region
	}
	return l, nil
}

func NewLocalNodeFromNodeProto(node *conkit.Node) (*LocalNodeImpl, error) {
	return &LocalNodeImpl{node: utils.CloneProto(node)}, nil
}

func (l *LocalNodeImpl) Clone() *conkit.Node {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return utils.CloneProto(l.node)
}

// for testing only
func (l *LocalNodeImpl) SetNodeID(nodeID conkit.NodeID) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.node.Id = string(nodeID)
}

func (l *LocalNodeImpl) NodeID() conkit.NodeID {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return conkit.NodeID(l.node.Id)
}

func (l *LocalNodeImpl) NodeType() conkit.NodeType {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.node.Type
}

func (l *LocalNodeImpl) NodeIP() string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.node.Ip
}

func (l *LocalNodeImpl) Region() string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.node.Region
}

func (l *LocalNodeImpl) SetState(state conkit.NodeState) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.node.State = state
}

// for testing only
func (l *LocalNodeImpl) SetStats(stats *conkit.NodeStats) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.node.Stats = utils.CloneProto(stats)
}

func (l *LocalNodeImpl) UpdateNodeStats() bool {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.prevStats == nil {
		l.prevStats = l.node.Stats
	}
	updated, computedAvg, err := prometheus.GetUpdatedNodeStats(l.node.Stats, l.prevStats)
	if err != nil {
		logger.Errorw("could not update node stats", err)
		return false
	}
	l.node.Stats = updated
	if computedAvg {
		l.prevStats = updated
	}
	return true
}

func (l *LocalNodeImpl) SecondsSinceNodeStatsUpdate() float64 {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return time.Since(time.Unix(l.node.Stats.UpdatedAt, 0)).Seconds()
}
