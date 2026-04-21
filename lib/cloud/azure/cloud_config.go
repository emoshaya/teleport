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
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	armcloud "github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"

	imdsazure "github.com/gravitational/teleport/lib/cloud/imds/azure"
)

const (
	azureAuthorityHostEnvVar = "AZURE_AUTHORITY_HOST"
	imdsLookupTimeout        = 2 * time.Second

	msGraphEndpointPublic = "https://graph.microsoft.com"
	msGraphEndpointChina  = "https://microsoftgraph.chinacloudapi.cn"
	msGraphEndpointUSGov  = "https://graph.microsoft.us"
)

type cloudConfig struct {
	authorityHost string
	audience      string
	endpoint      string
}

var (
	publicCloudConfig = cloudConfig{
		authorityHost: "https://login.microsoftonline.com/",
		audience:      "https://management.core.windows.net/",
		endpoint:      "https://management.azure.com",
	}
	chinaCloudConfig = cloudConfig{
		authorityHost: "https://login.chinacloudapi.cn/",
		audience:      "https://management.core.chinacloudapi.cn/",
		endpoint:      "https://management.chinacloudapi.cn",
	}
	governmentCloudConfig = cloudConfig{
		authorityHost: "https://login.microsoftonline.us/",
		audience:      "https://management.core.usgovcloudapi.net/",
		endpoint:      "https://management.usgovcloudapi.net",
	}
	germanCloudConfig = cloudConfig{
		authorityHost: "https://login.microsoftonline.de/",
		audience:      "https://management.core.cloudapi.de/",
		endpoint:      "https://management.microsoftazure.de",
	}
)

var cloudConfigByEnvironment = map[string]cloudConfig{
	normalizeCloudEnvironment(imdsazure.AzurePublicCloudEnvironment):       publicCloudConfig,
	normalizeCloudEnvironment(imdsazure.AzureChinaCloudEnvironment):        chinaCloudConfig,
	normalizeCloudEnvironment(imdsazure.AzureUSGovernmentEnvironment):      governmentCloudConfig,
	normalizeCloudEnvironment(imdsazure.AzureUSGovernmentCloudEnvironment): governmentCloudConfig,
	normalizeCloudEnvironment(imdsazure.AzureGermanCloudEnvironment):       germanCloudConfig,
}

var cloudConfigByAuthorityHost = map[string]cloudConfig{
	"login.microsoftonline.com":        publicCloudConfig,
	"login.windows.net":                publicCloudConfig,
	"login.chinacloudapi.cn":           chinaCloudConfig,
	"login.partner.microsoftonline.cn": chinaCloudConfig,
	"login.microsoftonline.us":         governmentCloudConfig,
	"login.microsoftonline.de":         germanCloudConfig,
}

var msGraphEndpointByEnvironment = map[string]string{
	normalizeCloudEnvironment(imdsazure.AzurePublicCloudEnvironment):       msGraphEndpointPublic,
	normalizeCloudEnvironment(imdsazure.AzureChinaCloudEnvironment):        msGraphEndpointChina,
	normalizeCloudEnvironment(imdsazure.AzureUSGovernmentEnvironment):      msGraphEndpointUSGov,
	normalizeCloudEnvironment(imdsazure.AzureUSGovernmentCloudEnvironment): msGraphEndpointUSGov,
	// Microsoft Graph Germany endpoint was retired. Fall back to the global endpoint.
	normalizeCloudEnvironment(imdsazure.AzureGermanCloudEnvironment): msGraphEndpointPublic,
}

var msGraphEndpointByAuthorityHost = map[string]string{
	"login.microsoftonline.com":        msGraphEndpointPublic,
	"login.windows.net":                msGraphEndpointPublic,
	"login.chinacloudapi.cn":           msGraphEndpointChina,
	"login.partner.microsoftonline.cn": msGraphEndpointChina,
	"login.microsoftonline.us":         msGraphEndpointUSGov,
	// Microsoft Graph Germany endpoint was retired. Fall back to the global endpoint.
	"login.microsoftonline.de": msGraphEndpointPublic,
}

var cloudConfigByAudience = map[string]cloudConfig{
	normalizeAudience("https://management.azure.com/"):              publicCloudConfig,
	normalizeAudience("https://management.core.windows.net/"):       publicCloudConfig,
	normalizeAudience("https://management.chinacloudapi.cn/"):       chinaCloudConfig,
	normalizeAudience("https://management.core.chinacloudapi.cn/"):  chinaCloudConfig,
	normalizeAudience("https://management.usgovcloudapi.net/"):      governmentCloudConfig,
	normalizeAudience("https://management.core.usgovcloudapi.net/"): governmentCloudConfig,
	normalizeAudience("https://management.microsoftazure.de/"):      germanCloudConfig,
	normalizeAudience("https://management.core.cloudapi.de/"):       germanCloudConfig,
}

var (
	defaultCloudConfigOnce sync.Once
	defaultCloudConfig     armcloud.Configuration
)

// GetARMClientOptions returns ARM client options configured for the inferred Azure cloud.
func GetARMClientOptions(ctx context.Context, cloudEnvironment string) *arm.ClientOptions {
	return &arm.ClientOptions{
		ClientOptions: GetClientOptions(ctx, cloudEnvironment),
	}
}

// GetClientOptions returns Azure core client options configured for the inferred Azure cloud.
func GetClientOptions(ctx context.Context, cloudEnvironment string) azcore.ClientOptions {
	return azcore.ClientOptions{
		Cloud: resolveCloudConfiguration(ctx, cloudEnvironment),
	}
}

// GetDefaultAzureCredentialOptions returns default credential options configured for the inferred Azure cloud.
func GetDefaultAzureCredentialOptions(ctx context.Context, cloudEnvironment string) *azidentity.DefaultAzureCredentialOptions {
	return &azidentity.DefaultAzureCredentialOptions{
		ClientOptions: GetClientOptions(ctx, cloudEnvironment),
	}
}

// GetClientAssertionCredentialOptions returns client assertion credential options configured for the inferred Azure cloud.
func GetClientAssertionCredentialOptions(ctx context.Context, cloudEnvironment string) *azidentity.ClientAssertionCredentialOptions {
	return &azidentity.ClientAssertionCredentialOptions{
		ClientOptions: GetClientOptions(ctx, cloudEnvironment),
	}
}

// GetMSGraphEndpoint returns a Microsoft Graph endpoint for the provided cloud
// environment. If cloudEnvironment is empty, it will attempt to infer it from
// AZURE_AUTHORITY_HOST or IMDS and falls back to the global endpoint.
func GetMSGraphEndpoint(ctx context.Context, cloudEnvironment string) string {
	if endpoint, ok := msGraphEndpointByEnvironment[normalizeCloudEnvironment(cloudEnvironment)]; ok {
		return endpoint
	}
	if endpoint, ok := msGraphEndpointByAuthorityHost[normalizeAuthorityHost(os.Getenv(azureAuthorityHostEnvVar))]; ok {
		return endpoint
	}
	if endpoint, ok := msGraphEndpointFromIMDS(ctx); ok {
		return endpoint
	}
	return msGraphEndpointPublic
}

// GetManagedIdentityCredentialOptions returns managed identity credential options configured for the inferred Azure cloud.
func GetManagedIdentityCredentialOptions(ctx context.Context, cloudEnvironment string, id azidentity.ManagedIDKind) *azidentity.ManagedIdentityCredentialOptions {
	return &azidentity.ManagedIdentityCredentialOptions{
		ClientOptions: GetClientOptions(ctx, cloudEnvironment),
		ID:            id,
	}
}

// GetWorkloadIdentityCredentialOptions returns workload identity credential options configured for the inferred Azure cloud.
func GetWorkloadIdentityCredentialOptions(ctx context.Context, cloudEnvironment, clientID string) *azidentity.WorkloadIdentityCredentialOptions {
	return &azidentity.WorkloadIdentityCredentialOptions{
		ClientOptions: GetClientOptions(ctx, cloudEnvironment),
		ClientID:      clientID,
	}
}

// MapWorkloadIdentityScope normalizes known Azure ARM resource-manager scopes for workload identity credentials.
func MapWorkloadIdentityScope(scope string) string {
	if strings.HasSuffix(scope, "/.default") {
		return scope
	}
	if _, ok := cloudConfigurationForAudience(scope); ok {
		return strings.TrimSuffix(scope, "/") + "/.default"
	}
	return scope
}

func resolveCloudConfiguration(ctx context.Context, cloudEnvironment string) armcloud.Configuration {
	if cfg, ok := cloudConfigurationForEnvironment(cloudEnvironment); ok {
		return cfg
	}
	if cfg, ok := cloudConfigurationForAuthorityHost(os.Getenv(azureAuthorityHostEnvVar)); ok {
		return cfg
	}
	return resolveDefaultCloudConfiguration(ctx)
}

func resolveDefaultCloudConfiguration(ctx context.Context) armcloud.Configuration {
	defaultCloudConfigOnce.Do(func() {
		if cfg, ok := cloudConfigurationFromIMDS(ctx); ok {
			defaultCloudConfig = cfg
			return
		}
		defaultCloudConfig = toSDKCloudConfiguration(publicCloudConfig)
	})
	return defaultCloudConfig
}

func cloudConfigurationFromIMDS(ctx context.Context) (armcloud.Configuration, bool) {
	if ctx == nil {
		ctx = context.Background()
	}

	imdsCtx := ctx
	cancelFn := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		imdsCtx, cancelFn = context.WithTimeout(ctx, imdsLookupTimeout)
	}
	defer cancelFn()

	info, err := imdsazure.NewInstanceMetadataClient().GetInstanceInfo(imdsCtx)
	if err != nil || info == nil {
		return armcloud.Configuration{}, false
	}
	return cloudConfigurationForEnvironment(info.CloudEnvironment)
}

func msGraphEndpointFromIMDS(ctx context.Context) (string, bool) {
	if ctx == nil {
		ctx = context.Background()
	}

	imdsCtx := ctx
	cancelFn := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		imdsCtx, cancelFn = context.WithTimeout(ctx, imdsLookupTimeout)
	}
	defer cancelFn()

	info, err := imdsazure.NewInstanceMetadataClient().GetInstanceInfo(imdsCtx)
	if err != nil || info == nil {
		return "", false
	}

	endpoint, ok := msGraphEndpointByEnvironment[normalizeCloudEnvironment(info.CloudEnvironment)]
	return endpoint, ok
}

func cloudConfigurationForEnvironment(cloudEnvironment string) (armcloud.Configuration, bool) {
	cfg, ok := cloudConfigByEnvironment[normalizeCloudEnvironment(cloudEnvironment)]
	if !ok {
		return armcloud.Configuration{}, false
	}
	return toSDKCloudConfiguration(cfg), true
}

func cloudConfigurationForAudience(audience string) (armcloud.Configuration, bool) {
	cfg, ok := cloudConfigByAudience[normalizeAudience(audience)]
	if !ok {
		return armcloud.Configuration{}, false
	}
	return toSDKCloudConfiguration(cfg), true
}

func cloudConfigurationForAuthorityHost(authorityHost string) (armcloud.Configuration, bool) {
	cfg, ok := cloudConfigByAuthorityHost[normalizeAuthorityHost(authorityHost)]
	if !ok {
		return armcloud.Configuration{}, false
	}
	return toSDKCloudConfiguration(cfg), true
}

func toSDKCloudConfiguration(cfg cloudConfig) armcloud.Configuration {
	return armcloud.Configuration{
		ActiveDirectoryAuthorityHost: cfg.authorityHost,
		Services: map[armcloud.ServiceName]armcloud.ServiceConfiguration{
			armcloud.ResourceManager: {
				Audience: cfg.audience,
				Endpoint: cfg.endpoint,
			},
		},
	}
}

func normalizeAudience(audience string) string {
	audience = strings.TrimSpace(audience)
	audience = strings.TrimSuffix(audience, "/.default")
	audience = strings.TrimSuffix(audience, "/")
	return strings.ToLower(audience)
}

func normalizeCloudEnvironment(cloudEnvironment string) string {
	return strings.ToLower(strings.TrimSpace(cloudEnvironment))
}

func normalizeAuthorityHost(authorityHost string) string {
	authorityHost = strings.TrimSpace(authorityHost)
	if authorityHost == "" {
		return ""
	}

	rawURL := authorityHost
	if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return strings.ToLower(strings.Trim(authorityHost, "/"))
	}
	if parsed.Hostname() != "" {
		return strings.ToLower(parsed.Hostname())
	}
	return strings.ToLower(strings.Trim(authorityHost, "/"))
}
