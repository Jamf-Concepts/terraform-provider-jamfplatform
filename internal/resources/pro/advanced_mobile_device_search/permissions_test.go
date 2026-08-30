// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package advanced_mobile_device_search

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// clientCallRe matches SDK method calls on the typed Pro client value, e.g.
// `r.client.GetAdvancedMobileDeviceSearchV1(` or `d.client.ListAdvancedMobileDeviceSearchesV1(`.
var clientCallRe = regexp.MustCompile(`\bclient\.([A-Za-z0-9]+)\(`)

// registryFilteredCalls reads the named source file and returns the set of
// client.<Method> calls it makes that are present in pro.Privileges. Filtering
// to the registry drops resolver wrappers (e.g.
// ResolveAdvancedMobileDeviceSearchV1ByName) that are not distinct privilege
// entries — they resolve-then-read and require only the read privilege the GET
// method already documents.
func registryFilteredCalls(t *testing.T, filename string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	called := map[string]bool{}
	for _, m := range clientCallRe.FindAllStringSubmatch(string(src), -1) {
		if _, known := pro.Privileges[m[1]]; known {
			called[m[1]] = true
		}
	}
	return called
}

// assertCallsMatch fails if the registry-backed client calls in filename differ
// from declared.
func assertCallsMatch(t *testing.T, filename string, declared []string) {
	t.Helper()
	called := registryFilteredCalls(t, filename)
	want := map[string]bool{}
	for _, m := range declared {
		want[m] = true
	}

	var undeclared, uncalled []string
	for m := range called {
		if !want[m] {
			undeclared = append(undeclared, m)
		}
	}
	for m := range want {
		if !called[m] {
			uncalled = append(uncalled, m)
		}
	}
	sort.Strings(undeclared)
	sort.Strings(uncalled)

	if len(undeclared) > 0 {
		t.Errorf("%s calls SDK methods missing from declared list: %v", filename, undeclared)
	}
	if len(uncalled) > 0 {
		t.Errorf("declared list contains methods %s does not call: %v", filename, uncalled)
	}
}

// TestResourceSDKMethods_KnownToSDK fails if a declared resource method has been
// renamed or removed in the SDK privilege registry.
func TestResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, resourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("resourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestResourceSDKMethods_MatchCRUDCalls keeps resourceSDKMethods in sync with the
// client.<Method> calls in crud.go.
func TestResourceSDKMethods_MatchCRUDCalls(t *testing.T) {
	assertCallsMatch(t, "crud.go", resourceSDKMethods)
}

// TestDataSourceSDKMethods_KnownToSDK fails if a declared data source method has
// been renamed or removed in the SDK privilege registry.
func TestDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, dataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("dataSourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestDataSourceSDKMethods_MatchReadCalls keeps dataSourceSDKMethods in sync with
// the registry-backed client.<Method> calls in data_source.go.
func TestDataSourceSDKMethods_MatchReadCalls(t *testing.T) {
	assertCallsMatch(t, "data_source.go", dataSourceSDKMethods)
}

// TestListResourceSDKMethods_KnownToSDK fails if a declared list resource method
// has been renamed or removed in the SDK privilege registry.
func TestListResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, listResourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("listResourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestListResourceSDKMethods_MatchListCalls keeps listResourceSDKMethods in sync
// with the client.<Method> calls in list_resource.go.
func TestListResourceSDKMethods_MatchListCalls(t *testing.T) {
	assertCallsMatch(t, "list_resource.go", listResourceSDKMethods)
}

// TestPrivileges_Rendered guards that each table actually rendered the
// advanced-mobile-device-searches privileges (catches an empty/parse-skipped
// registry).
func TestPrivileges_Rendered(t *testing.T) {
	for name, got := range map[string]string{
		"resourcePrivileges":     resourcePrivileges,
		"dataSourcePrivileges":   dataSourcePrivileges,
		"listResourcePrivileges": listResourcePrivileges,
	} {
		if !strings.Contains(got, "advanced-device-searches:") {
			t.Errorf("%s did not render the advanced-mobile-device-searches privileges:\n%s", name, got)
		}
	}
}
