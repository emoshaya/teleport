/**
 * Copyright (C) 2025 Gravitational, Inc.
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

package recordingmetadata

import (
	"context"
	"time"

	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/session"
)

type SessionType int

const (
	SessionTypeUnspecified SessionType = iota
	SessionTypeTTY
)

// Service defines an interface for processing session recordings.
type Service interface {
	// ProcessSessionRecording processes the session recording associated with the
	// provided session ID.
	ProcessSessionRecording(ctx context.Context, sessionID session.ID, sessionType SessionType, duration time.Duration) error
}

// ProcessingInput groups the inputs to [Service.ProcessSessionRecording].
type ProcessingInput struct {
	// SessionType is the session recording type (e.g. TTY, desktop).
	SessionType SessionType
	// Duration is the session duration.
	Duration time.Duration
}

// InputFromSessionEnd derives a [ProcessingInput] from a session end event.
// Returns nil if sessionEnd is nil, if the event type has no corresponding
// [SessionType], or if its timestamps are unset.
//
// When adding a new SessionType, also:
//   - add a corresponding case below,
//   - update TestInputFromSessionEnd in service_test.go,
//   - update TestRecordingMetadata_StreamDispatch in lib/events/stream_test.go.
func InputFromSessionEnd(sessionEnd apievents.AuditEvent) *ProcessingInput {
	var sessionType SessionType
	var start, end time.Time
	switch e := sessionEnd.(type) {
	case *apievents.SessionEnd:
		sessionType = SessionTypeTTY
		start, end = e.StartTime, e.EndTime
	default:
		return nil
	}

	if start.IsZero() || end.IsZero() {
		return nil
	}

	return &ProcessingInput{
		SessionType: sessionType,
		Duration:    end.Sub(start),
	}
}
