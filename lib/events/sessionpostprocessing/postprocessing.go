/*
 * Teleport
 * Copyright (C) 2025  Gravitational, Inc.
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

package sessionpostprocessing

import (
	"context"

	"github.com/gravitational/trace"

	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/auth/recordingmetadata"
	"github.com/gravitational/teleport/lib/auth/summarizer"
	"github.com/gravitational/teleport/lib/session"
)

// Config is the configuration for the session post-processor.
type Config struct {
	// SessionSummarizerProvider is a provider of the session summarizer service.
	// It can be nil or provide a nil summarizer if summarization is not needed.
	// The summarizer itself summarizes session recordings.
	SessionSummarizerProvider *summarizer.SessionSummarizerProvider
	// RecordingMetadataProvider is a provider of the recording metadata service.
	// When nil, recording-metadata processing is skipped.
	RecordingMetadataProvider *recordingmetadata.Provider
	// SessionEnd is the session end event to process. Nil is valid for sessions
	// that terminated without an end event (e.g. aborted uploads); in that case
	// [summarizer.SessionSummarizer.SummarizeWithoutEndEvent] is used.
	SessionEnd apievents.AuditEvent
	// SessionID is the ID of the session being processed.
	SessionID session.ID
	// MetadataFallback, when set, is used for recording-metadata processing
	// if SessionEnd is missing or lacks timestamps. Callers that track their
	// own start/end times (e.g. the streaming writer) can supply this so
	// metadata is still produced from a truncated stream.
	MetadataFallback *recordingmetadata.ProcessingInput
}

// Process processes session end events after the session recording upload is complete.
// It summarizes the session recording and processes the recording metadata.
func Process(ctx context.Context, cfg Config) error {
	var summarizerErr, metadataErr error

	switch {
	case cfg.SessionSummarizerProvider == nil:
		return trace.BadParameter("session summarizer provider is not set")
	case cfg.SessionID == "":
		return trace.BadParameter("session ID is not set")
	}

	if cfg.RecordingMetadataProvider != nil {
		metadata := recordingmetadata.InputFromSessionEnd(cfg.SessionEnd)
		if metadata == nil {
			metadata = cfg.MetadataFallback
		}
		if metadata != nil {
			metadataSvc := cfg.RecordingMetadataProvider.Service()
			if err := metadataSvc.ProcessSessionRecording(ctx, cfg.SessionID, metadata.SessionType, metadata.Duration); err != nil {
				metadataErr = trace.Wrap(err, "failed to process session recording metadata")
			}
		}
	}

	sessionSummarizer := cfg.SessionSummarizerProvider.SessionSummarizer()

	switch end := cfg.SessionEnd.(type) {
	case *apievents.SessionEnd:
		if err := sessionSummarizer.SummarizeSSH(ctx, end); err != nil {
			summarizerErr = trace.Wrap(err, "failed to summarize upload")
		}

	case *apievents.DatabaseSessionEnd:
		if err := sessionSummarizer.SummarizeDatabase(ctx, end); err != nil {
			summarizerErr = trace.Wrap(err, "failed to summarize upload")
		}

	case nil:
		if err := sessionSummarizer.SummarizeWithoutEndEvent(ctx, cfg.SessionID); err != nil {
			summarizerErr = trace.Wrap(err, "failed to summarize upload")
		}
	}

	return trace.NewAggregate(summarizerErr, metadataErr)
}
