/*
 * Teleport
 * Copyright (C) 2026  Gravitational, Inc.
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

package azurejoin

import "testing"

func TestIssuerMatchesAzureTenant(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		issuerURL string
		tenantID  string
		want      bool
	}{
		{
			name:      "sts windows v1 issuer",
			issuerURL: "https://sts.windows.net/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      true,
		},
		{
			name:      "china partner v2 issuer",
			issuerURL: "https://login.partner.microsoftonline.cn/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/v2.0",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      true,
		},
		{
			name:      "german issuer",
			issuerURL: "https://login.microsoftonline.de/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      true,
		},
		{
			name:      "unsupported issuer host",
			issuerURL: "https://example.com/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      false,
		},
		{
			name:      "issuer tenant mismatch",
			issuerURL: "https://sts.windows.net/11111111-2222-3333-4444-555555555555/",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      false,
		},
		{
			name:      "non-https issuer",
			issuerURL: "http://sts.windows.net/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee/",
			tenantID:  "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
			want:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := issuerMatchesAzureTenant(tc.issuerURL, tc.tenantID)
			if got != tc.want {
				t.Fatalf("issuerMatchesAzureTenant(%q, %q) = %v, want %v", tc.issuerURL, tc.tenantID, got, tc.want)
			}
		})
	}
}

func TestIsAzureResourceManagerAudience(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		audience string
		want     bool
	}{
		{
			name:     "public arm",
			audience: "https://management.azure.com/",
			want:     true,
		},
		{
			name:     "china arm",
			audience: "https://management.chinacloudapi.cn/",
			want:     true,
		},
		{
			name:     "usgov arm",
			audience: "https://management.usgovcloudapi.net/",
			want:     true,
		},
		{
			name:     "german arm",
			audience: "https://management.microsoftazure.de/",
			want:     true,
		},
		{
			name:     "unknown audience",
			audience: "https://graph.microsoft.com/",
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isAzureResourceManagerAudience(tc.audience)
			if got != tc.want {
				t.Fatalf("isAzureResourceManagerAudience(%q) = %v, want %v", tc.audience, got, tc.want)
			}
		})
	}
}
