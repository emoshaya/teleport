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

package types

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gravitational/trace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJamfSpecV1(t *testing.T) {
	validSpec := &JamfSpecV1{
		Enabled:     true,
		ApiEndpoint: "https://yourtenant.jamfcloud.com",
	}
	validEntry := &JamfInventoryEntry{
		FilterRsql:        "", // no filters
		SyncPeriodPartial: 0,  // default period
		SyncPeriodFull:    0,  // default period
		OnMissing:         "", // same as NOOP
	}

	modify := func(f func(spec *JamfSpecV1)) *JamfSpecV1 {
		spec := proto.Clone(validSpec).(*JamfSpecV1)
		f(spec)
		return spec
	}

	tests := []struct {
		name    string
		spec    *JamfSpecV1
		wantErr string
	}{
		{
			name: "minimal spec",
			spec: validSpec,
		},
		{
			name: "spec with inventory",
			spec: &JamfSpecV1{
				Enabled:     true,
				ApiEndpoint: "https://yourtenant.jamfcloud.com",
				Inventory: []*JamfInventoryEntry{
					{
						FilterRsql:        `general.remoteManagement.managed==true and general.platform=="Mac"`,
						SyncPeriodPartial: Duration(4 * time.Hour),
						SyncPeriodFull:    Duration(48 * time.Hour),
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
			spec: modify(func(spec *JamfSpecV1) {
				spec.ApiEndpoint = "https://%%"
			}),
			wantErr: "API endpoint",
		},
		{
			name: "api_endpoint empty hostname",
			spec: modify(func(spec *JamfSpecV1) {
				spec.ApiEndpoint = "not a valid URL"
			}),
			wantErr: "missing hostname",
		},
		{
			name: "inventory nil entry",
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
					nil,
				}
			}),
			wantErr: "is nil",
		},
		{
			name: "inventory sync_partial > sync_full",
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: Duration(12 * time.Hour),
						SyncPeriodFull:    Duration(8 * time.Hour),
					},
				}
			}),
			wantErr: "greater or equal to sync_period_full",
		},
		{
			name: "inventory on_missing invalid",
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
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
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: -1,
						SyncPeriodFull:    Duration(8 * time.Hour),
					},
				}
			}),
		},
		{
			name: "inventory sync_full disabled",
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
					validEntry,
					{
						SyncPeriodPartial: Duration(12 * time.Hour),
						SyncPeriodFull:    -1,
					},
				}
			}),
		},
		{
			name: "inventory all syncs disabled",
			spec: modify(func(spec *JamfSpecV1) {
				spec.Inventory = []*JamfInventoryEntry{
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
			err := ValidateJamfSpecV1(test.spec)
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
	want := &JamfSpecV1{
		ApiEndpoint: "https://test.jamfcloud.com",
		SyncDelay:   Duration(6 * time.Hour),
		Inventory: []*JamfInventoryEntry{
			{
				FilterRsql:        "general.remoteManagement.managed==true",
				SyncPeriodPartial: Duration(6 * time.Hour),
				SyncPeriodFull:    Duration(24 * time.Hour),
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
			var got JamfSpecV1
			err := got.UnmarshalJSON([]byte(tt.json))
			require.NoError(t, err)
			require.Equal(t, want, &got)
		})
	}
}

func TestJamfKeyRenamesMatchStructTags(t *testing.T) {
	t.Run("JamfSpecV1", func(t *testing.T) {
		checkKeyRenames(t, reflect.TypeFor[JamfSpecV1](), jamfSpecV1KeyRenames)
	})
	t.Run("JamfInventoryEntry", func(t *testing.T) {
		checkKeyRenames(t, reflect.TypeFor[JamfInventoryEntry](), jamfInventoryEntryKeyRenames)
	})
}

// checkKeyRenames verifies that the given rename map covers all struct fields
// where the jsonpb camelCase name differs from the encoding/json snake_case
// name. If a field is missing from the map, the test fails.
func checkKeyRenames(t *testing.T, structType reflect.Type, renames map[string]string) {
	t.Helper()
	for i := range structType.NumField() {
		f := structType.Field(i)
		if strings.HasPrefix(f.Name, "XXX_") {
			continue
		}

		camelName := protoJSONName(f.Tag.Get("protobuf"))
		snakeName := structTagJSONName(f.Tag.Get("json"))
		if camelName == "" || snakeName == "" || snakeName == "-" {
			continue
		}
		if camelName == snakeName {
			continue
		}

		assert.Contains(t, renames, camelName,
			"field %s has jsonpb name %q and json tag name %q but is missing from "+
				"the key rename map; update the map so UnmarshalJSONPB can handle "+
				"this field correctly", f.Name, camelName, snakeName)
	}
}

// protoJSONName extracts the JSON field name from a protobuf struct tag.
// For example, from `protobuf:"varint,3,opt,name=sync_delay,json=syncDelay,proto3"`,
// it returns "syncDelay". Returns "" if no json= option is present.
func protoJSONName(tag string) string {
	for part := range strings.SplitSeq(tag, ",") {
		if v, ok := strings.CutPrefix(part, "json="); ok {
			return v
		}
	}
	return ""
}

// structTagJSONName extracts the field name from a json struct tag.
// For example, from `json:"sync_delay,omitempty"`, it returns "sync_delay".
func structTagJSONName(tag string) string {
	name, _, _ := strings.Cut(tag, ",")
	return name
}
