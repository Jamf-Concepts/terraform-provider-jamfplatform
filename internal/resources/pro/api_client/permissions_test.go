// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package api_client

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// clientCallRe matches SDK client method calls of the form `client.<Method>(`
// (covering `r.client.<Method>(` and `d.client.<Method>(`).
var clientCallRe = regexp.MustCompile(`\bclient\.([A-Za-z0-9]+)\(`)

// calledSDKMethods returns the distinct SDK method names called in the named
// source file that are present in the pro privilege registry. Filtering to
// known registry methods keeps non-SDK `client.` lookalikes out of the set.
func calledSDKMethods(t *testing.T, filename string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	called := map[string]bool{}
	for _, m := range clientCallRe.FindAllStringSubmatch(string(src), -1) {
		if _, ok := pro.Privileges[m[1]]; ok {
			called[m[1]] = true
		}
	}
	return called
}

// assertMatch fails if the source file calls an SDK method not declared, or
// declares one it does not call.
func assertMatch(t *testing.T, filename string, declaredList []string) {
	t.Helper()
	called := calledSDKMethods(t, filename)
	declared := map[string]bool{}
	for _, m := range declaredList {
		declared[m] = true
	}

	var undeclared, uncalled []string
	for m := range called {
		if !declared[m] {
			undeclared = append(undeclared, m)
		}
	}
	for m := range declared {
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

// --- resource (crud.go) ---

// TestResourceSDKMethods_KnownToSDK fails if a declared method has been renamed
// or removed in the SDK privilege registry.
func TestResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, resourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("resourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestResourceSDKMethods_MatchCRUDCalls keeps the resource privileges table in
// sync with the actual client.<Method> calls in crud.go.
func TestResourceSDKMethods_MatchCRUDCalls(t *testing.T) {
	assertMatch(t, "crud.go", resourceSDKMethods)
}

// TestResourcePrivileges_Rendered guards that the table actually rendered.
func TestResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(resourcePrivileges, "create:pro:api-integrations") {
		t.Fatalf("resourcePrivileges did not render the api-integrations privileges:\n%s", resourcePrivileges)
	}
}

// --- data source (data_source.go) ---

// TestDataSourceSDKMethods_KnownToSDK fails if a declared method has been
// renamed or removed in the SDK privilege registry.
func TestDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, dataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("dataSourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestDataSourceSDKMethods_MatchCalls keeps the data source privileges table in
// sync with the actual client.<Method> calls in data_source.go.
func TestDataSourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "data_source.go", dataSourceSDKMethods)
}

// TestDataSourcePrivileges_Rendered guards that the table actually rendered.
func TestDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(dataSourcePrivileges, "read:pro:api-integrations") {
		t.Fatalf("dataSourcePrivileges did not render the api-integrations privileges:\n%s", dataSourcePrivileges)
	}
}

// --- plural data source (datasource_plural.go) ---

// TestPluralDataSourceSDKMethods_KnownToSDK fails if a declared method has been
// renamed or removed in the SDK privilege registry.
func TestPluralDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, pluralDataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("pluralDataSourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestPluralDataSourceSDKMethods_MatchCalls keeps the plural data source
// privileges table in sync with the calls in datasource_plural.go.
func TestPluralDataSourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "datasource_plural.go", pluralDataSourceSDKMethods)
}

// TestPluralDataSourcePrivileges_Rendered guards that the table rendered.
func TestPluralDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(pluralDataSourcePrivileges, "read:pro:api-integrations") {
		t.Fatalf("pluralDataSourcePrivileges did not render the api-integrations privileges:\n%s", pluralDataSourcePrivileges)
	}
}

// --- list resource (list_resource.go) ---

// TestListResourceSDKMethods_KnownToSDK fails if a declared method has been
// renamed or removed in the SDK privilege registry.
func TestListResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, listResourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("listResourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

// TestListResourceSDKMethods_MatchCalls keeps the list resource privileges
// table in sync with the calls in list_resource.go.
func TestListResourceSDKMethods_MatchCalls(t *testing.T) {
	assertMatch(t, "list_resource.go", listResourceSDKMethods)
}

// TestListResourcePrivileges_Rendered guards that the table rendered.
func TestListResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(listResourcePrivileges, "read:pro:api-integrations") {
		t.Fatalf("listResourcePrivileges did not render the api-integrations privileges:\n%s", listResourcePrivileges)
	}
}
