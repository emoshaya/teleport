/*
Copyright 2024 Gravitational, Inc.

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
package entraid

import (
	"net/url"
	"os"
	"path"
	"strings"
)

// FederationMetadataURL returns the URL for the federation metadata endpoint
func FederationMetadataURL(tenantID, appID string) string {
	const defaultHost = "login.microsoftonline.com"

	host := defaultHost
	if value := normalizeAuthorityHost(os.Getenv("AZURE_AUTHORITY_HOST")); value != "" {
		switch value {
		case "login.microsoftonline.com", "login.windows.net", "login.microsoftonline.us", "login.chinacloudapi.cn", "login.partner.microsoftonline.cn":
			host = value
		}
	}

	return (&url.URL{
		Scheme: "https",
		Host:   host,
		Path:   path.Join(tenantID, "federationmetadata", "2007-06", "federationmetadata.xml"),
		RawQuery: url.Values{
			"appid": {appID},
		}.Encode(),
	}).String()
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
	if err == nil && parsed.Hostname() != "" {
		return strings.ToLower(parsed.Hostname())
	}

	return strings.ToLower(strings.Trim(authorityHost, "/"))
}
