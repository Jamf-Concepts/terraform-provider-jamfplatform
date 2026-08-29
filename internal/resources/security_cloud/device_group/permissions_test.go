// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package device_group

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// clientCallRe matches SDK client method calls of the form
// <receiver>.client.<Method>( — covering r.client / d.client receivers used by
// the resource, data sources, and list resource in this package.
var clientCallRe = regexp.MustCompile(`\bclient\.([A-Za-z0-9]+)\(`)

// calledMethods returns the distinct SDK client method names invoked in the given
// source file, restricted to methods known to the SDK privilege registry so
// unrelated identifiers (helpers, framework calls) are ignored.
//
// Sibling packages carry a syntheticMethodBacking map here, mapping a
// Resolve<X>ByName / Apply<X> helper onto the generated method whose privileges it
// consumes, because the registry omits those by design. This package calls no such
// helper — the singular data source matches names over ListDeviceGroupsV2 itself —
// so there is nothing to map. Reintroducing a synthetic call without the map is
// still caught: the registry does not carry it, so the declared list would report
// ListDeviceGroupsV2 as uncalled.
func calledMethods(t *testing.T, filename string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	called := map[string]bool{}
	for _, m := range clientCallRe.FindAllStringSubmatch(string(src), -1) {
		name := m[1]
		if _, ok := securitycloud.Privileges[name]; ok {
			called[name] = true
		}
	}
	return called
}

// assertMatch fails if the called set and the declared set differ.
func assertMatch(t *testing.T, filename string, declared []string) {
	t.Helper()
	called := calledMethods(t, filename)
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
		t.Errorf("%s calls SDK methods missing from the declared list: %v", filename, undeclared)
	}
	if len(uncalled) > 0 {
		t.Errorf("declared list has methods %s does not call: %v", filename, uncalled)
	}
}

// --- resource ---

// TestResourceSDKMethods_KnownToSDK fails if a declared method has been renamed
// or removed in the SDK privilege registry.
func TestResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(securitycloud.Privileges, resourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("resourceSDKMethods not present in securitycloud.Privileges (SDK drift): %v", missing)
	}
}

// TestResourceSDKMethods_MatchCRUDCalls keeps the resource privileges table
// honest as the CRUD path changes.
func TestResourceSDKMethods_MatchCRUDCalls(t *testing.T) {
	assertMatch(t, "crud.go", resourceSDKMethods)
}

// TestResourcePrivileges_Rendered guards that the table actually rendered into
// the resource description.
func TestResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(resourcePrivileges, "device-groups:create") {
		t.Fatalf("resourcePrivileges did not render the device group privileges:\n%s", resourcePrivileges)
	}
}

// --- data source ---

func TestDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(securitycloud.Privileges, dataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("dataSourceSDKMethods not present in securitycloud.Privileges (SDK drift): %v", missing)
	}
}

func TestDataSourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "data_source.go", dataSourceSDKMethods)
}

func TestDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(dataSourcePrivileges, "device-groups:read") {
		t.Fatalf("dataSourcePrivileges did not render the device group privileges:\n%s", dataSourcePrivileges)
	}
}

// --- plural data source ---

func TestPluralDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(securitycloud.Privileges, pluralDataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("pluralDataSourceSDKMethods not present in securitycloud.Privileges (SDK drift): %v", missing)
	}
}

func TestPluralDataSourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "datasource_plural.go", pluralDataSourceSDKMethods)
}

func TestPluralDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(pluralDataSourcePrivileges, "device-groups:read") {
		t.Fatalf("pluralDataSourcePrivileges did not render the device group privileges:\n%s", pluralDataSourcePrivileges)
	}
}

// --- list resource ---

func TestListResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(securitycloud.Privileges, listResourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("listResourceSDKMethods not present in securitycloud.Privileges (SDK drift): %v", missing)
	}
}

func TestListResourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "list_resource.go", listResourceSDKMethods)
}

func TestListResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(listResourcePrivileges, "device-groups:read") {
		t.Fatalf("listResourcePrivileges did not render the device group privileges:\n%s", listResourcePrivileges)
	}
}
