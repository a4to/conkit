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

package selector_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/a4to/protocol/conkit"

	"github.com/a4to/conkit-server/pkg/routing/selector"
)

var (
	nodeLoadLow = &conkit.Node{
		State: conkit.NodeState_SERVING,
		Stats: &conkit.NodeStats{
			UpdatedAt:       time.Now().Unix(),
			NumCpus:         1,
			CpuLoad:         0.1,
			LoadAvgLast1Min: 0.0,
			NumRooms: 1,
			NumClients: 2,
			NumTracksIn: 4,
			NumTracksOut: 8,
			BytesInPerSec: 1000,
			BytesOutPerSec: 2000,
		},
	}

	nodeLoadMedium = &conkit.Node{
		State: conkit.NodeState_SERVING,
		Stats: &conkit.NodeStats{
			UpdatedAt:       time.Now().Unix(),
			NumCpus:         1,
			CpuLoad:         0.5,
			LoadAvgLast1Min: 0.5,
			NumRooms: 5,
			NumClients: 10,
			NumTracksIn: 20,
			NumTracksOut: 200,
			BytesInPerSec: 5000,
			BytesOutPerSec: 10000,
		},
	}

	nodeLoadHigh = &conkit.Node{
		State: conkit.NodeState_SERVING,
		Stats: &conkit.NodeStats{
			UpdatedAt:       time.Now().Unix(),
			NumCpus:         1,
			CpuLoad:         0.99,
			LoadAvgLast1Min: 2.0,
			NumRooms: 10,
			NumClients: 20,
			NumTracksIn: 40,
			NumTracksOut: 800,
			BytesInPerSec: 10000,
			BytesOutPerSec: 40000,
		},
	}
)

func TestSystemLoadSelector_SelectNode(t *testing.T) {
	sel := selector.SystemLoadSelector{SysloadLimit: 1.0, SortBy: "random"}

	var nodes []*conkit.Node
	_, err := sel.SelectNode(nodes)
	require.Error(t, err, "should error no available nodes")

	// Select a node with high load when no nodes with low load are available
	nodes = []*conkit.Node{nodeLoadHigh}
	if _, err := sel.SelectNode(nodes); err != nil {
		t.Error(err)
	}

	// Select a node with low load when available
	nodes = []*conkit.Node{nodeLoadLow, nodeLoadHigh}
	for i := 0; i < 5; i++ {
		node, err := sel.SelectNode(nodes)
		if err != nil {
			t.Error(err)
		}
		if node != nodeLoadLow {
			t.Error("selected the wrong node")
		}
	}
}
