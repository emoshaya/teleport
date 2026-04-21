package entraid

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFederationMetadataURLAuthorityHost(t *testing.T) {
	t.Setenv("AZURE_AUTHORITY_HOST", "")
	defaultURL, err := url.Parse(FederationMetadataURL("tenant-id", "app-id"))
	require.NoError(t, err)
	require.Equal(t, "login.microsoftonline.com", defaultURL.Host)

	t.Setenv("AZURE_AUTHORITY_HOST", "https://login.chinacloudapi.cn/")
	chinaURL, err := url.Parse(FederationMetadataURL("tenant-id", "app-id"))
	require.NoError(t, err)
	require.Equal(t, "login.chinacloudapi.cn", chinaURL.Host)

	t.Setenv("AZURE_AUTHORITY_HOST", "login.microsoftonline.us")
	usGovURL, err := url.Parse(FederationMetadataURL("tenant-id", "app-id"))
	require.NoError(t, err)
	require.Equal(t, "login.microsoftonline.us", usGovURL.Host)
}
