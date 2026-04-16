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
	"encoding/json"
	"net/url"
	"slices"
	"strings"

	"github.com/gogo/protobuf/jsonpb" //nolint:depguard // needed for backwards compatibility
	"github.com/gravitational/trace"
)

const (
	// JamfOnMissingNOOP is the textual representation for the NOOP on_missing
	// action.
	JamfOnMissingNoop = "NOOP"
	// JamfOnMissingDelete is the textual representation for the DELETE on_missing
	// action.
	JamfOnMissingDelete = "DELETE"
)

// JamfOnMissingActions is a slice of all textual on_missing representations,
// excluding the empty string.
var JamfOnMissingActions = []string{
	JamfOnMissingNoop,
	JamfOnMissingDelete,
}

// ValidateJamfSpecV1 validates a [JamfSpecV1] instance.
func ValidateJamfSpecV1(s *JamfSpecV1) error {
	if s == nil {
		return trace.BadParameter("spec required")
	}

	switch u, err := url.Parse(s.ApiEndpoint); {
	case err != nil:
		return trace.BadParameter("invalid API endpoint: %v", err)
	case u.Host == "":
		return trace.BadParameter("invalid API endpoint: missing hostname")
	}

	for i, e := range s.Inventory {
		switch {
		case e == nil:
			return trace.BadParameter("inventory entry #%v is nil", i)
		case e.OnMissing != "" && !slices.Contains(JamfOnMissingActions, e.OnMissing):
			return trace.BadParameter(
				"inventory[%v]: invalid on_missing action %q (expect empty or one of [%v])",
				i, e.OnMissing, strings.Join(JamfOnMissingActions, ","))
		}

		syncPartial := e.SyncPeriodPartial
		syncFull := e.SyncPeriodFull
		if syncFull > 0 && syncPartial >= syncFull {
			return trace.BadParameter("inventory[%v]: sync_period_partial is greater or equal to sync_period_full, partial syncs will never happen", i)
		}
	}

	return nil
}

// jamfSpecV1KeyRenames maps camelCase JSON keys (produced by gogoproto's jsonpb
// marshaler) to the snake_case keys expected by the struct's json tags.
var jamfSpecV1KeyRenames = map[string]string{
	"syncDelay":   "sync_delay",
	"apiEndpoint": "api_endpoint",
}

// jamfInventoryEntryKeyRenames maps camelCase JSON keys to snake_case keys for
// JamfInventoryEntry fields.
var jamfInventoryEntryKeyRenames = map[string]string{
	"filterRsql":        "filter_rsql",
	"syncPeriodPartial": "sync_period_partial",
	"syncPeriodFull":    "sync_period_full",
	"onMissing":         "on_missing",
	"pageSize":          "page_size",
}

// UnmarshalJSONPB implements jsonpb.JSONPBUnmarshaler for JamfSpecV1.
//
// gogoproto's jsonpb has a bug where it calls Duration.MarshalJSON on int64
// casttype Duration fields (producing strings like "6h0m0s") but fails to call
// Duration.UnmarshalJSON during unmarshal, instead stripping quotes and passing
// invalid bare text to json.Unmarshal. Implementing JSONPBUnmarshaler bypasses
// this broken path entirely.
// See https://github.com/gravitational/teleport/issues/57747.
func (s *JamfSpecV1) UnmarshalJSONPB(_ *jsonpb.Unmarshaler, data []byte) error {
	return s.unmarshalJSON(data)
}

// UnmarshalJSON implements json.Unmarshaler for JamfSpecV1. This is called by
// encoding/json when processing inventory entries inside JamfSpecV1 (since
// UnmarshalJSONPB delegates to encoding/json).
func (s *JamfSpecV1) UnmarshalJSON(data []byte) error {
	return s.unmarshalJSON(data)
}

func (s *JamfSpecV1) unmarshalJSON(data []byte) error {
	data, err := renameJSONKeys(data, jamfSpecV1KeyRenames)
	if err != nil {
		return trace.Wrap(err)
	}
	// Type alias prevents json.Unmarshal from calling our UnmarshalJSON
	// recursively, while still calling Duration.UnmarshalJSON on Duration fields.
	type raw JamfSpecV1
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return trace.Wrap(err)
	}
	*s = JamfSpecV1(v)
	return nil
}

// UnmarshalJSONPB implements jsonpb.JSONPBUnmarshaler for JamfInventoryEntry.
// See JamfSpecV1.UnmarshalJSONPB for details on why this is needed.
func (e *JamfInventoryEntry) UnmarshalJSONPB(_ *jsonpb.Unmarshaler, data []byte) error {
	return e.unmarshalJSON(data)
}

// UnmarshalJSON implements json.Unmarshaler for JamfInventoryEntry. This is
// called by encoding/json when processing inventory entries inside JamfSpecV1
// (since JamfSpecV1.UnmarshalJSONPB delegates to encoding/json).
func (e *JamfInventoryEntry) UnmarshalJSON(data []byte) error {
	return e.unmarshalJSON(data)
}

func (e *JamfInventoryEntry) unmarshalJSON(data []byte) error {
	data, err := renameJSONKeys(data, jamfInventoryEntryKeyRenames)
	if err != nil {
		return trace.Wrap(err)
	}
	// Type alias prevents json.Unmarshal from calling our UnmarshalJSON
	// recursively, while still calling Duration.UnmarshalJSON on Duration fields.
	type raw JamfInventoryEntry
	var v raw
	if err := json.Unmarshal(data, &v); err != nil {
		return trace.Wrap(err)
	}
	*e = JamfInventoryEntry(v)
	return nil
}

// renameJSONKeys renames keys in a JSON object according to the given map.
// Keys not in the map are left unchanged. If the target key already exists, the
// rename is skipped to avoid overwriting.
func renameJSONKeys(data []byte, renames map[string]string) ([]byte, error) {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, trace.Wrap(err)
	}
	changed := false
	for from, to := range renames {
		val, ok := m[from]
		if !ok {
			continue
		}
		if _, exists := m[to]; exists {
			continue
		}
		m[to] = val
		delete(m, from)
		changed = true
	}
	if !changed {
		return data, nil
	}
	return json.Marshal(m)
}
