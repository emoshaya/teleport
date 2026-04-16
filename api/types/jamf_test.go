// Copyright 2023 Gravitational, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types_test

import (
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gravitational/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gravitational/teleport/api/types"
)

func TestValidateJamfSpecV1(t *testing.T) {
	validSpec := &types.JamfSpecV1{
		Enabled:     true,
		ApiEndpoint: "https://yourtenant.jamfcloud.com",
	}
	validEntry := &types.JamfInventoryEntry{
		FilterRsql:        "", // no filters
		SyncPeriodPartial: 0,  // default period
		SyncPeriodFull:    0,  // default period
		OnMissing:         "", // same as NOOP
	}

	modify := func(f func(spec *types.JamfSpecV1)) *types.JamfSpecV1 {
		spec := proto.Clone(validSpec).(*types.JamfSpecV1)
		f(spec)
		return spec
	}

	tests := []struct {
		name    string
		spec    *types.JamfSpecV1
		wantErr string
	}{
		{
			name: "minimal spec",
			spec: validSpec,
		},
		{
			name: "spec with inventory",
			spec: &types.JamfSpecV1{
				Enabled:     true,
				ApiEndpoint: "https://yourtenant.jamfcloud.com",
				Inventory: []*types.JamfInventoryEntry{
					{
						FilterRsql:        `general.remoteManagement.managed==true and general.platform=="Mac"`,
						SyncPeriodPartial: types.Duration(4 * time.Hour),
						SyncPeriodFull:    types.Duration(48 * time.Hour),
						OnMissing:         "DELETE",
					},
					{
						FilterRsql: `general.remoteManagement.managed==false`,
						OnMissing:  "NOOP",
					},
					validEntry,
				},
			},
		},
		{
			name:    "nil spec",
			spec:    nil,
			wantErr: "spec required",
		},
		{
			name: "api_endpoint invalid",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.ApiEndpoint = "https://%%"
			}),
			wantErr: "API endpoint",
		},
		{
			name: "api_endpoint empty hostname",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.ApiEndpoint = "not a valid URL"
			}),
			wantErr: "missing hostname",
		},
		{
			name: "inventory nil entry",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					nil,
				}
			}),
			wantErr: "is nil",
		},
		{
			name: "inventory sync_partial > sync_full",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: types.Duration(12 * time.Hour),
						SyncPeriodFull:    types.Duration(8 * time.Hour),
					},
				}
			}),
			wantErr: "greater or equal to sync_period_full",
		},
		{
			name: "inventory on_missing invalid",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					validEntry,
					{
						OnMissing: "BANANA",
					},
				}
			}),
			wantErr: "on_missing",
		},
		{
			name: "inventory sync_partial disabled",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: -1,
						SyncPeriodFull:    types.Duration(8 * time.Hour),
					},
				}
			}),
		},
		{
			name: "inventory sync_full disabled",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: types.Duration(12 * time.Hour),
						SyncPeriodFull:    -1,
					},
				}
			}),
		},
		{
			name: "inventory all syncs disabled",
			spec: modify(func(spec *types.JamfSpecV1) {
				spec.Inventory = []*types.JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: 0,
						SyncPeriodFull:    0,
					},
				}
			}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := types.ValidateJamfSpecV1(test.spec)
			if test.wantErr == "" {
				assert.NoError(t, err, "ValidateJamfSpecV1 failed")
			} else {
				assert.ErrorContains(t, err, test.wantErr, "ValidateJamfSpecV1 error mismatch")
				assert.True(t, trace.IsBadParameter(err), "ValidateJamfSpecV1 returned non-BadParameter error: %T", err)
			}
		})
	}
}

func TestJamfSpecV1UnmarshalJSON(t *testing.T) {
	want := &types.JamfSpecV1{
		ApiEndpoint: "https://test.jamfcloud.com",
		SyncDelay:   types.Duration(6 * time.Hour),
		Inventory: []*types.JamfInventoryEntry{
			{
				FilterRsql:        "general.remoteManagement.managed==true",
				SyncPeriodPartial: types.Duration(6 * time.Hour),
				SyncPeriodFull:    types.Duration(24 * time.Hour),
				OnMissing:         "DELETE",
				PageSize:          50,
			},
		},
	}

	tests := []struct {
		name string
		json string
	}{
		{
			name: "camelCase keys (from jsonpb marshal)",
			json: `{
				"syncDelay": "6h0m0s",
				"apiEndpoint": "https://test.jamfcloud.com",
				"inventory": [{
					"filterRsql": "general.remoteManagement.managed==true",
					"syncPeriodPartial": "6h0m0s",
					"syncPeriodFull": "24h0m0s",
					"onMissing": "DELETE",
					"pageSize": 50
				}]
			}`,
		},
		{
			name: "snake_case keys (from json struct tags)",
			json: `{
				"sync_delay": "6h0m0s",
				"api_endpoint": "https://test.jamfcloud.com",
				"inventory": [{
					"filter_rsql": "general.remoteManagement.managed==true",
					"sync_period_partial": "6h0m0s",
					"sync_period_full": "24h0m0s",
					"on_missing": "DELETE",
					"page_size": 50
				}]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got types.JamfSpecV1
			err := got.UnmarshalJSON([]byte(tt.json))
			require.NoError(t, err)
			require.Equal(t, want, &got)
		})
	}
}
