/**
 * Copyright (C) 2026 Gravitational, Inc.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package recordingmetadata_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/auth/recordingmetadata"
)

func TestInputFromSessionEnd(t *testing.T) {
	start := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)
	end := start.Add(5 * time.Minute)

	tests := []struct {
		name  string
		event apievents.AuditEvent
		want  *recordingmetadata.ProcessingInput
	}{
		{
			name: "SSH SessionEnd with valid timestamps",
			event: &apievents.SessionEnd{
				StartTime: start,
				EndTime:   end,
			},
			want: &recordingmetadata.ProcessingInput{
				SessionType: recordingmetadata.SessionTypeTTY,
				Duration:    5 * time.Minute,
			},
		},
		{
			name:  "SSH SessionEnd with zero StartTime",
			event: &apievents.SessionEnd{EndTime: end},
			want:  nil,
		},
		{
			name:  "SSH SessionEnd with zero EndTime",
			event: &apievents.SessionEnd{StartTime: start},
			want:  nil,
		},
		{
			name:  "DatabaseSessionEnd — update when SessionTypeDatabase is added",
			event: &apievents.DatabaseSessionEnd{StartTime: start, EndTime: end},
			want:  nil,
		},
		{
			name:  "WindowsDesktopSessionEnd — update when SessionTypeDesktop is added",
			event: &apievents.WindowsDesktopSessionEnd{StartTime: start, EndTime: end},
			want:  nil,
		},
		{
			name:  "nil event",
			event: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordingmetadata.InputFromSessionEnd(tt.event)

			require.Equal(t, tt.want, got)
		})
	}
}
