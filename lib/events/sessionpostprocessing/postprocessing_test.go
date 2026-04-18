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
package sessionpostprocessing_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	apievents "github.com/gravitational/teleport/api/types/events"
	"github.com/gravitational/teleport/lib/auth/recordingmetadata"
	"github.com/gravitational/teleport/lib/auth/summarizer"
	"github.com/gravitational/teleport/lib/events/eventstest"
	"github.com/gravitational/teleport/lib/events/sessionpostprocessing"
	"github.com/gravitational/teleport/lib/session"
)

// TestProcessRecoversFromPanic ensures Process turns a panic in any downstream
// component (summarizer, recording metadata, etc.) into a regular error
// return. The gRPC handler that calls Process only logs errors, so this is
// what keeps a misbehaving recording (e.g. corrupt data from a self-hosted
// S3 clone driving vt10x into a bad state) from crashing auth.
func TestProcessRecoversFromPanic(t *testing.T) {
	sessionID := session.ID(uuid.NewString())
	events := eventstest.GenerateTestSession(eventstest.SessionParams{
		UserName:  "alice",
		SessionID: string(sessionID),
		ServerID:  "testcluster",
		PrintData: []string{"boom"},
	})
	sessionEnd := events[len(events)-1]

	tests := []struct {
		name        string
		summarizer  summarizer.SessionSummarizer
		metadata    recordingmetadata.Service // nil leaves NewProvider's default no-op in place
		wantMessage string
	}{
		{
			name:        "metadata panics",
			summarizer:  summarizer.NoopSummarizer{},
			metadata:    &panickingRecordingMetadata{},
			wantMessage: "simulated recording metadata panic",
		},
		{
			name:        "summarizer panics",
			summarizer:  &panickingSummarizer{},
			wantMessage: "simulated summarizer panic",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			metadataProvider := recordingmetadata.NewProvider()
			if tc.metadata != nil {
				metadataProvider.SetService(tc.metadata)
			}

			summarizerProvider := summarizer.NewSessionSummarizerProvider()
			summarizerProvider.SetSummarizer(tc.summarizer)

			err := sessionpostprocessing.Process(t.Context(), sessionpostprocessing.Config{
				SessionEnd:                sessionEnd,
				RecordingMetadataProvider: metadataProvider,
				SessionSummarizerProvider: summarizerProvider,
				SessionID:                 sessionID,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantMessage)
		})
	}
}

type panickingRecordingMetadata struct{}

func (p *panickingRecordingMetadata) ProcessSessionRecording(context.Context, session.ID, recordingmetadata.SessionType, time.Duration) error {
	panic("simulated recording metadata panic")
}

type panickingSummarizer struct{}

func (p *panickingSummarizer) SummarizeSSH(context.Context, *apievents.SessionEnd) error {
	panic("simulated summarizer panic")
}

func (p *panickingSummarizer) SummarizeDatabase(context.Context, *apievents.DatabaseSessionEnd) error {
	panic("simulated summarizer panic")
}

func (p *panickingSummarizer) SummarizeWithoutEndEvent(context.Context, session.ID) error {
	panic("simulated summarizer panic")
}

func TestSessionPostProcessor(t *testing.T) {
	sessionID := session.ID(uuid.NewString())

	metadataProvider := recordingmetadata.NewProvider()
	recorderMetadata := &fakeRecordingMetadata{}
	recorderMetadata.On(
		"ProcessSessionRecording",
		mock.Anything,
		sessionID,
		mock.Anything,
		mock.Anything,
	).
		Return(nil).Once()
	metadataProvider.SetService(recorderMetadata)

	summarizerProvider := summarizer.NewSessionSummarizerProvider()
	sessionSummarizer := &fakeSummarizer{}
	sessionSummarizer.On(
		"SummarizeSSH",
		mock.Anything,
		mock.Anything,
	).Return(nil).Once()
	summarizerProvider.SetSummarizer(sessionSummarizer)

	events := eventstest.GenerateTestSession(eventstest.SessionParams{
		UserName:  "alice",
		SessionID: string(sessionID),
		ServerID:  "testcluster",
		PrintData: []string{"net", "stat"},
	})

	cfg := sessionpostprocessing.Config{
		SessionEnd:                events[len(events)-1],
		RecordingMetadataProvider: metadataProvider,
		SessionSummarizerProvider: summarizerProvider,
		SessionID:                 sessionID,
	}

	err := sessionpostprocessing.Process(t.Context(), cfg)
	require.NoError(t, err)

	recorderMetadata.AssertExpectations(t)
	sessionSummarizer.AssertExpectations(t)
}

type fakeRecordingMetadata struct {
	mock.Mock
}

func (f *fakeRecordingMetadata) ProcessSessionRecording(ctx context.Context, sessionID session.ID, sessionType recordingmetadata.SessionType, duration time.Duration) error {
	args := f.Called(ctx, sessionID, sessionType, duration)
	return args.Error(0)
}

type fakeSummarizer struct {
	mock.Mock
}

func (f *fakeSummarizer) SummarizeSSH(ctx context.Context, sessionEndEvent *apievents.SessionEnd) error {
	args := f.Called(ctx, sessionEndEvent)
	return args.Error(0)
}

func (f *fakeSummarizer) SummarizeDatabase(ctx context.Context, sessionEndEvent *apievents.DatabaseSessionEnd) error {
	args := f.Called(ctx, sessionEndEvent)
	return args.Error(0)
}

func (f *fakeSummarizer) SummarizeWithoutEndEvent(ctx context.Context, sessionID session.ID) error {
	args := f.Called(ctx, sessionID)
	return args.Error(0)
}
