// Copyright 2022 Gravitational, Inc
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

package azure

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsMSSQLServerEndpoint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc     string
		endpoint string
		result   bool
	}{
		// valid
		{"only suffix", ".database.windows.net", true},
		{"suffix with port", ".database.windows.net:1604", true},
		{"full name", "random.database.windows.net:1604", true},
		{"china full name", "random.database.chinacloudapi.cn:1604", true},
		{"usgov full name", "random.database.usgovcloudapi.net:1604", true},
		// invalid
		{"empty", "", false},
		{"without suffix", "hello:1604", false},
		{"wrong suffix", "hello.database.azure.com:1604", false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.result, IsMSSQLServerEndpoint(tc.endpoint))
		})
	}
}

func TestIsDatabaseEndpoint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc     string
		endpoint string
		result   bool
	}{
		{"public mysql", "server.mysql.database.azure.com:3306", true},
		{"china mysql", "server.mysql.database.chinacloudapi.cn:3306", true},
		{"usgov postgres", "server.postgres.database.usgovcloudapi.net:5432", true},
		{"china mssql", "server.database.chinacloudapi.cn:1433", false},
		{"empty", "", false},
		{"random", "example.com:3306", false},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.result, IsDatabaseEndpoint(tc.endpoint))
		})
	}
}

func TestParseMSSQLEndpoint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc     string
		endpoint string
		valid    bool
		name     string
	}{
		// valid
		{"valid", "random.database.windows.net:1604", true, "random"},
		{"china", "random.database.chinacloudapi.cn:1604", true, "random"},
		{"usgov", "random.database.usgovcloudapi.net:1604", true, "random"},
		// invalid
		{"empty", "", false, ""},
		{"malformed address", "abc", false, ""},
		{"only suffix", ".database.windows.net:1604", false, ""},
		{"without suffix", "example.com:1604", false, ""},
		{"without port", "random.database.windows.net", false, ""},
		{"without port china", "random.database.chinacloudapi.cn", false, ""},
		{"wrong suffix", "random.database.azure.com:1604", false, ""},
		{"more segments than supported", "hello.random.database.windows.net:1604", false, ""},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			name, err := ParseMSSQLEndpoint(tc.endpoint)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tc.name, name)
		})
	}
}

func TestParseDatabaseEndpoint(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc     string
		endpoint string
		valid    bool
		name     string
	}{
		{"public mysql", "random.mysql.database.azure.com:3306", true, "random"},
		{"china mysql", "random.mysql.database.chinacloudapi.cn:3306", true, "random"},
		{"usgov postgres", "random.postgres.database.usgovcloudapi.net:5432", true, "random"},
		{"china mssql should fail", "random.database.chinacloudapi.cn:1433", false, ""},
		{"without port", "random.mysql.database.azure.com", false, ""},
		{"invalid suffix", "random.mysql.example.com:3306", false, ""},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			name, err := ParseDatabaseEndpoint(tc.endpoint)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			require.Equal(t, tc.name, name)
		})
	}
}

func TestGetOSSRDBMSAADTokenScope(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		desc     string
		endpoint string
		scope    string
	}{
		{
			desc:     "public",
			endpoint: "random.mysql.database.azure.com:3306",
			scope:    OSSRDBMSAADScopePublic,
		},
		{
			desc:     "china",
			endpoint: "random.mysql.database.chinacloudapi.cn:3306",
			scope:    OSSRDBMSAADScopeChina,
		},
		{
			desc:     "usgov",
			endpoint: "random.postgres.database.usgovcloudapi.net:5432",
			scope:    OSSRDBMSAADScopeUSGov,
		},
		{
			desc:     "url-form endpoint",
			endpoint: "https://random.mysql.database.chinacloudapi.cn",
			scope:    OSSRDBMSAADScopeChina,
		},
		{
			desc:     "unknown defaults public",
			endpoint: "example.com:443",
			scope:    OSSRDBMSAADScopePublic,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			require.Equal(t, tc.scope, GetOSSRDBMSAADTokenScope(tc.endpoint))
		})
	}
}

func TestIsAzureEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		hostname string
		want     bool
	}{
		{
			name:     "empty",
			hostname: "",
			want:     false,
		},
		{
			name:     "valid endpoint",
			hostname: "management.azure.com",
			want:     true,
		},
		{
			name:     "valid endpoint prefix",
			hostname: "subdomain.core.windows.net",
			want:     true,
		},
		{
			name:     "invalid endpoint, with valid prefix",
			hostname: "core.windows.net.example.com",
			want:     false,
		},
		{
			name:     "invalid endpoint",
			hostname: "not-azure.example.com",
			want:     false,
		},
		{
			name:     "china management endpoint",
			hostname: "management.chinacloudapi.cn",
			want:     true,
		},
		{
			name:     "china login endpoint",
			hostname: "login.chinacloudapi.cn",
			want:     true,
		},
		{
			name:     "usgov sql endpoint suffix",
			hostname: "db.database.usgovcloudapi.net",
			want:     true,
		},
		{
			name:     "china storage endpoint suffix",
			hostname: "account.core.chinacloudapi.cn",
			want:     true,
		},
		{
			name:     "invalid endpoint, suffix match without dot",
			hostname: "my-azurefd.net",
			want:     false,
		},
		{
			name:     "valid endpoint, suffix matches with dot",
			hostname: "my.azurefd.net",
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsAzureEndpoint(tt.hostname))
		})
	}
}
