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

package azure

import (
	"context"
	"testing"

	armcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/stretchr/testify/require"

	imdsazure "github.com/gravitational/teleport/lib/cloud/imds/azure"
)

func TestGetClientOptions_CloudEnvironment(t *testing.T) {
	opts := GetClientOptions(context.Background(), imdsazure.AzureChinaCloudEnvironment)
	require.Equal(t, "https://login.chinacloudapi.cn/", opts.Cloud.ActiveDirectoryAuthorityHost)

	rmCfg, ok := opts.Cloud.Services[armcloud.ResourceManager]
	require.True(t, ok)
	require.Equal(t, "https://management.core.chinacloudapi.cn/", rmCfg.Audience)
	require.Equal(t, "https://management.chinacloudapi.cn", rmCfg.Endpoint)
}

func TestCloudConfigurationForAuthorityHost(t *testing.T) {
	cfg, ok := cloudConfigurationForAuthorityHost("https://login.partner.microsoftonline.cn/")
	require.True(t, ok)
	require.Equal(t, "https://login.chinacloudapi.cn/", cfg.ActiveDirectoryAuthorityHost)

	rmCfg, ok := cfg.Services[armcloud.ResourceManager]
	require.True(t, ok)
	require.Equal(t, "https://management.core.chinacloudapi.cn/", rmCfg.Audience)
	require.Equal(t, "https://management.chinacloudapi.cn", rmCfg.Endpoint)
}

func TestMapWorkloadIdentityScope(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		expected string
	}{
		{
			name:     "public core arm audience",
			in:       "https://management.core.windows.net/",
			expected: "https://management.core.windows.net/.default",
		},
		{
			name:     "china arm endpoint audience",
			in:       "https://management.chinacloudapi.cn/",
			expected: "https://management.chinacloudapi.cn/.default",
		},
		{
			name:     "already normalized",
			in:       "https://management.core.chinacloudapi.cn/.default",
			expected: "https://management.core.chinacloudapi.cn/.default",
		},
		{
			name:     "other scope unchanged",
			in:       "some-other-scope",
			expected: "some-other-scope",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, MapWorkloadIdentityScope(tc.in))
		})
	}
}

func TestGetMSGraphEndpoint_CloudEnvironment(t *testing.T) {
	require.Equal(t, msGraphEndpointChina, GetMSGraphEndpoint(context.Background(), imdsazure.AzureChinaCloudEnvironment))
	require.Equal(t, msGraphEndpointUSGov, GetMSGraphEndpoint(context.Background(), imdsazure.AzureUSGovernmentCloudEnvironment))
	require.Equal(t, msGraphEndpointPublic, GetMSGraphEndpoint(context.Background(), imdsazure.AzurePublicCloudEnvironment))
}

func TestGetMSGraphEndpoint_AuthorityHost(t *testing.T) {
	t.Setenv(azureAuthorityHostEnvVar, "https://login.partner.microsoftonline.cn/")
	require.Equal(t, msGraphEndpointChina, GetMSGraphEndpoint(context.Background(), ""))

	t.Setenv(azureAuthorityHostEnvVar, "login.microsoftonline.us")
	require.Equal(t, msGraphEndpointUSGov, GetMSGraphEndpoint(context.Background(), ""))
}
