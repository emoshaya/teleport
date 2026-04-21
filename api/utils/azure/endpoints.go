/*
Copyright 2022 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azure

import (
	"net"
	"net/url"
	"strings"

	"github.com/gravitational/trace"
)

// IsAzureEndpoint returns true if the input URI is an Azure endpoint.
//
// The code implements approximate solution based on:
// - https://management.azure.com/metadata/endpoints?api-version=2019-05-01
// - https://github.com/Azure/azure-cli/blob/dev/src/azure-cli-core/azure/cli/core/cloud.py
func IsAzureEndpoint(hostname string) bool {
	hostname = strings.ToLower(strings.TrimSpace(hostname))
	for _, suffix := range azureEndpointSuffixes {
		// exact match
		if hostname == suffix {
			return true
		}
		// .suffix match
		if strings.HasSuffix(hostname, "."+suffix) {
			return true
		}
	}

	return false
}

// IsDatabaseEndpoint returns true if provided endpoint is a valid database
// endpoint.
func IsDatabaseEndpoint(endpoint string) bool {
	host := normalizeEndpointHost(endpoint)
	for _, suffix := range databaseEndpointSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// IsCacheForRedisEndpoint returns true if provided endpoint is a valid Azure
// Cache for Redis endpoint.
func IsCacheForRedisEndpoint(endpoint string) bool {
	return IsRedisEndpoint(endpoint) || IsRedisEnterpriseEndpoint(endpoint)
}

// IsRedisEndpoint returns true if provided endpoint is a valid Redis
// (non-Enterprise tier) endpoint.
func IsRedisEndpoint(endpoint string) bool {
	return strings.Contains(endpoint, RedisEndpointSuffix)
}

// IsRedisEnterpriseEndpoint returns true if provided endpoint is a valid Redis
// Enterprise endpoint.
func IsRedisEnterpriseEndpoint(endpoint string) bool {
	return strings.Contains(endpoint, RedisEnterpriseEndpointSuffix)
}

// IsMSSQLServerEndpoint returns true if provided endpoint is a valid SQL server
// database endpoint.
func IsMSSQLServerEndpoint(endpoint string) bool {
	host := normalizeEndpointHost(endpoint)
	for _, suffix := range mssqlEndpointSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// ParseDatabaseEndpoint extracts database server name from Azure endpoint.
func ParseDatabaseEndpoint(endpoint string) (name string, err error) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", trace.Wrap(err)
	}
	host = strings.ToLower(host)
	// Azure endpoint looks like this:
	// name.mysql.database.azure.com
	parts := strings.Split(host, ".")
	if !hasDatabaseEndpointSuffix(host) || len(parts) != 5 {
		return "", trace.BadParameter("failed to parse %v as Azure endpoint", endpoint)
	}
	return parts[0], nil
}

// GetOSSRDBMSAADTokenScope returns the Azure OSS RDBMS AAD scope for a database endpoint.
// Defaults to public cloud scope when endpoint cloud cannot be inferred.
func GetOSSRDBMSAADTokenScope(endpoint string) string {
	host := normalizeEndpointHost(endpoint)
	if strings.HasSuffix(host, ".database.chinacloudapi.cn") {
		return OSSRDBMSAADScopeChina
	}
	if strings.HasSuffix(host, ".database.usgovcloudapi.net") {
		return OSSRDBMSAADScopeUSGov
	}
	return OSSRDBMSAADScopePublic
}

// ParseCacheForRedisEndpoint extracts database server name from Azure Cache
// for Redis endpoint.
func ParseCacheForRedisEndpoint(endpoint string) (name string, err error) {
	// Note that the Redis URI may contain schema and parameters.
	host, err := GetHostFromRedisURI(endpoint)
	if err != nil {
		return "", trace.Wrap(err)
	}

	switch {
	// Redis (non-Enterprise) endpoint looks like this:
	// name.redis.cache.windows.net
	case strings.HasSuffix(host, RedisEndpointSuffix):
		return strings.TrimSuffix(host, RedisEndpointSuffix), nil

	// Redis Enterprise endpoint looks like this:
	// name.region.redisenterprise.cache.azure.net
	case strings.HasSuffix(host, RedisEnterpriseEndpointSuffix):
		name, _, ok := strings.Cut(strings.TrimSuffix(host, RedisEnterpriseEndpointSuffix), ".")
		if !ok {
			return "", trace.BadParameter("failed to parse %v as Azure Cache endpoint", endpoint)
		}
		return name, nil

	default:
		return "", trace.BadParameter("failed to parse %v as Azure Cache endpoint", endpoint)
	}
}

// GetHostFromRedisURI extracts host name from a Redis URI. The URI may start
// with "redis://", "rediss://", or without. The URI may also have parameters
// like "?mode=cluster".
func GetHostFromRedisURI(uri string) (string, error) {
	// Add a temporary schema to make a valid URL for url.Parse if schema is
	// not found.
	if !strings.Contains(uri, "://") {
		uri = "schema://" + uri
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "", trace.Wrap(err)
	}
	return parsed.Hostname(), nil
}

// ParseMSSQLEndpoint extracts database server name from Azure endpoint.
func ParseMSSQLEndpoint(endpoint string) (name string, err error) {
	host, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return "", trace.Wrap(err)
	}
	// Azure endpoint looks like this:
	// name.database.windows.net
	host = strings.ToLower(host)
	parts := strings.Split(host, ".")
	if !hasMSSQLEndpointSuffix(host) || len(parts) != 4 {
		return "", trace.BadParameter("failed to parse %v as Azure MSSQL endpoint", endpoint)
	}

	if parts[0] == "" {
		return "", trace.BadParameter("endpoint %v must contain database name", endpoint)
	}

	return parts[0], nil
}

func hasDatabaseEndpointSuffix(host string) bool {
	for _, suffix := range databaseEndpointSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

func normalizeEndpointHost(endpoint string) string {
	host := strings.ToLower(strings.TrimSpace(endpoint))
	if host == "" {
		return host
	}

	// Accept URL-form and host[:port]-form endpoint strings.
	if strings.Contains(host, "://") {
		parsed, err := url.Parse(host)
		if err == nil && parsed != nil && parsed.Hostname() != "" {
			return strings.ToLower(parsed.Hostname())
		}
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		return strings.ToLower(parsedHost)
	}
	return host
}

const (
	// DatabaseEndpointSuffix is the Azure database endpoint suffix. Used for
	// MySQL, PostgreSQL, etc.
	DatabaseEndpointSuffix = ".database.azure.com"

	// OSSRDBMSAADScopePublic is the Azure OSS RDBMS AAD scope for public cloud.
	OSSRDBMSAADScopePublic = "https://ossrdbms-aad.database.windows.net/.default"
	// OSSRDBMSAADScopeChina is the Azure OSS RDBMS AAD scope for Azure China.
	OSSRDBMSAADScopeChina = "https://ossrdbms-aad.database.chinacloudapi.cn/.default"
	// OSSRDBMSAADScopeUSGov is the Azure OSS RDBMS AAD scope for Azure US Government.
	OSSRDBMSAADScopeUSGov = "https://ossrdbms-aad.database.usgovcloudapi.net/.default"

	// RedisEndpointSuffix is the endpoint suffix for Redis.
	RedisEndpointSuffix = ".redis.cache.windows.net"

	// RedisEnterpriseEndpointSuffix is the endpoint suffix for Redis Enterprise.
	RedisEnterpriseEndpointSuffix = ".redisenterprise.cache.azure.net"

	// MSSQLEndpointSuffix is the Azure SQL Server endpoint suffix.
	MSSQLEndpointSuffix = ".database.windows.net"
	// MSSQLEndpointSuffixChina is the Azure China SQL Server endpoint suffix.
	MSSQLEndpointSuffixChina = ".database.chinacloudapi.cn"
	// MSSQLEndpointSuffixUSGov is the Azure US Government SQL Server endpoint suffix.
	MSSQLEndpointSuffixUSGov = ".database.usgovcloudapi.net"
)

var databaseEndpointSuffixes = []string{
	// Public cloud supports mysql/postgres/mariadb under this suffix.
	DatabaseEndpointSuffix,
	// Azure China.
	".mysql.database.chinacloudapi.cn",
	".postgres.database.chinacloudapi.cn",
	".mariadb.database.chinacloudapi.cn",
	// Azure US Government.
	".mysql.database.usgovcloudapi.net",
	".postgres.database.usgovcloudapi.net",
	".mariadb.database.usgovcloudapi.net",
}

var mssqlEndpointSuffixes = []string{
	MSSQLEndpointSuffix,
	MSSQLEndpointSuffixChina,
	MSSQLEndpointSuffixUSGov,
}

var azureEndpointSuffixes = []string{
	// Resource manager / control plane endpoints.
	"management.azure.com",
	"management.chinacloudapi.cn",
	"management.usgovcloudapi.net",
	"management.core.windows.net",
	"management.core.chinacloudapi.cn",
	"management.core.usgovcloudapi.net",

	// Graph and auth endpoints.
	"graph.windows.net",
	"graph.chinacloudapi.cn",
	"graph.microsoftazure.us",
	"graph.microsoft.com",
	"microsoftgraph.chinacloudapi.cn",
	"graph.microsoft.us",
	"login.microsoftonline.com", // required for "az logout"
	"login.chinacloudapi.cn",
	"login.microsoftonline.us",

	// Common Azure service endpoints.
	"batch.core.windows.net",
	"batch.chinacloudapi.cn",
	"batch.core.usgovcloudapi.net",
	"rest.media.azure.net",
	"rest.media.chinacloudapi.cn",
	"rest.media.usgovcloudapi.net",
	"datalake.azure.net",
	"gallery.azure.com",
	"gallery.chinacloudapi.cn",
	"gallery.usgovcloudapi.net",
	"azuredatalakestore.net",
	"azuredatalakeanalytics.net",
	"azurecr.io",
	"azurecr.cn",
	"azurecr.us",
	"database.windows.net",
	"database.chinacloudapi.cn",
	"database.usgovcloudapi.net",
	"vault.azure.net",
	"vault.azure.cn",
	"vault.usgovcloudapi.net",
	"core.windows.net",
	"core.chinacloudapi.cn",
	"core.usgovcloudapi.net",
	"azurefd.net",
}

func hasMSSQLEndpointSuffix(host string) bool {
	for _, suffix := range mssqlEndpointSuffixes {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}
