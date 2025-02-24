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

package buffer

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/a4to/protocol/conkit"
)

func TestRidConversion(t *testing.T) {
	type RidAndLayer struct {
		rid   string
		layer int32
	}
	tests := []struct {
		name       string
		trackInfo  *conkit.TrackInfo
		ridToLayer map[string]RidAndLayer
	}{
		{
			"no track info",
			nil,
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: FullResolution, layer: 2},
			},
		},
		{
			"no layers",
			&conkit.TrackInfo{},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: FullResolution, layer: 2},
			},
		},
		{
			"single layer, low",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: QuarterResolution, layer: 0},
				FullResolution:    {rid: QuarterResolution, layer: 0},
			},
		},
		{
			"single layer, medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: QuarterResolution, layer: 0},
				FullResolution:    {rid: QuarterResolution, layer: 0},
			},
		},
		{
			"single layer, high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: QuarterResolution, layer: 0},
				FullResolution:    {rid: QuarterResolution, layer: 0},
			},
		},
		{
			"two layers, low and medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: HalfResolution, layer: 1},
			},
		},
		{
			"two layers, low and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: HalfResolution, layer: 1},
			},
		},
		{
			"two layers, medium and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: HalfResolution, layer: 1},
			},
		},
		{
			"three layers",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[string]RidAndLayer{
				"":                {rid: QuarterResolution, layer: 0},
				QuarterResolution: {rid: QuarterResolution, layer: 0},
				HalfResolution:    {rid: HalfResolution, layer: 1},
				FullResolution:    {rid: FullResolution, layer: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for testRid, expectedResult := range test.ridToLayer {
				actualLayer := RidToSpatialLayer(testRid, test.trackInfo)
				require.Equal(t, expectedResult.layer, actualLayer)

				actualRid := SpatialLayerToRid(actualLayer, test.trackInfo)
				require.Equal(t, expectedResult.rid, actualRid)
			}
		})
	}
}

func TestQualityConversion(t *testing.T) {
	type QualityAndLayer struct {
		quality conkit.VideoQuality
		layer   int32
	}
	tests := []struct {
		name           string
		trackInfo      *conkit.TrackInfo
		qualityToLayer map[conkit.VideoQuality]QualityAndLayer
	}{
		{
			"no track info",
			nil,
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 1},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 2},
			},
		},
		{
			"no layers",
			&conkit.TrackInfo{},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 1},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 2},
			},
		},
		{
			"single layer, low",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_LOW, layer: 0},
			},
		},
		{
			"single layer, medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_MEDIUM, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 0},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_MEDIUM, layer: 0},
			},
		},
		{
			"single layer, high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_HIGH, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_HIGH, layer: 0},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 0},
			},
		},
		{
			"two layers, low and medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 1},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_MEDIUM, layer: 1},
			},
		},
		{
			"two layers, low and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_HIGH, layer: 1},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 1},
			},
		},
		{
			"two layers, medium and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_MEDIUM, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 0},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 1},
			},
		},
		{
			"three layers",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]QualityAndLayer{
				conkit.VideoQuality_LOW:    {quality: conkit.VideoQuality_LOW, layer: 0},
				conkit.VideoQuality_MEDIUM: {quality: conkit.VideoQuality_MEDIUM, layer: 1},
				conkit.VideoQuality_HIGH:   {quality: conkit.VideoQuality_HIGH, layer: 2},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for testQuality, expectedResult := range test.qualityToLayer {
				actualLayer := VideoQualityToSpatialLayer(testQuality, test.trackInfo)
				require.Equal(t, expectedResult.layer, actualLayer)

				actualQuality := SpatialLayerToVideoQuality(actualLayer, test.trackInfo)
				require.Equal(t, expectedResult.quality, actualQuality)
			}
		})
	}
}

func TestVideoQualityToRidConversion(t *testing.T) {
	tests := []struct {
		name         string
		trackInfo    *conkit.TrackInfo
		qualityToRid map[conkit.VideoQuality]string
	}{
		{
			"no track info",
			nil,
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: HalfResolution,
				conkit.VideoQuality_HIGH:   FullResolution,
			},
		},
		{
			"no layers",
			&conkit.TrackInfo{},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: HalfResolution,
				conkit.VideoQuality_HIGH:   FullResolution,
			},
		},
		{
			"single layer, low",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: QuarterResolution,
				conkit.VideoQuality_HIGH:   QuarterResolution,
			},
		},
		{
			"single layer, medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: QuarterResolution,
				conkit.VideoQuality_HIGH:   QuarterResolution,
			},
		},
		{
			"single layer, high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: QuarterResolution,
				conkit.VideoQuality_HIGH:   QuarterResolution,
			},
		},
		{
			"two layers, low and medium",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: HalfResolution,
				conkit.VideoQuality_HIGH:   HalfResolution,
			},
		},
		{
			"two layers, low and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: HalfResolution,
				conkit.VideoQuality_HIGH:   HalfResolution,
			},
		},
		{
			"two layers, medium and high",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: QuarterResolution,
				conkit.VideoQuality_HIGH:   HalfResolution,
			},
		},
		{
			"three layers",
			&conkit.TrackInfo{
				Layers: []*conkit.VideoLayer{
					{Quality: conkit.VideoQuality_LOW},
					{Quality: conkit.VideoQuality_MEDIUM},
					{Quality: conkit.VideoQuality_HIGH},
				},
			},
			map[conkit.VideoQuality]string{
				conkit.VideoQuality_LOW:    QuarterResolution,
				conkit.VideoQuality_MEDIUM: HalfResolution,
				conkit.VideoQuality_HIGH:   FullResolution,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for testQuality, expectedRid := range test.qualityToRid {
				actualRid := VideoQualityToRid(testQuality, test.trackInfo)
				require.Equal(t, expectedRid, actualRid)
			}
		})
	}
}
