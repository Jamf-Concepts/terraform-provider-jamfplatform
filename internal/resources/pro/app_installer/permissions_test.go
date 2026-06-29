// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/permissions"
)

// clientCallRE matches an SDK client method call on the resource/data-source
// receiver (r.client.Foo( / d.client.Foo( / client.Foo(); the trailing "(" and
// the \b anchor on "client" keep it to genuine call sites.
var clientCallRE = regexp.MustCompile(`\bclient\.([A-Za-z0-9]+)\(`)

// callsIn returns the distinct SDK client method names called in the named
// source file, filtered to those present in pro.Privileges. Filtering to the
// registry drops convenience wrappers (e.g. the list-backed Resolve…ByName
// resolver) that have no own privilege entry — their underlying request is
// already covered by a tracked GET/List.
func callsIn(t *testing.T, filename string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	called := map[string]bool{}
	for _, m := range clientCallRE.FindAllStringSubmatch(string(src), -1) {
		if _, ok := pro.Privileges[m[1]]; ok {
			called[m[1]] = true
		}
	}
	return called
}

// assertMatch fails when the registry-known SDK calls discovered in filename do
// not exactly equal declared.
func assertMatch(t *testing.T, filename string, declared []string) {
	t.Helper()
	called := callsIn(t, filename)
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

// --- Resource (crud.go) ---

func TestResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, resourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("resourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

func TestResourceSDKMethods_MatchCRUDCalls(t *testing.T) {
	assertMatch(t, "crud.go", resourceSDKMethods)
}

func TestResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(resourcePrivileges, "create:pro:mac-applications") {
		t.Fatalf("resourcePrivileges did not render the App Installer privileges:\n%s", resourcePrivileges)
	}
}

// --- Singular data source (data_source.go) ---

func TestDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, dataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("dataSourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

func TestDataSourceSDKMethods_MatchReadCalls(t *testing.T) {
	assertMatch(t, "data_source.go", dataSourceSDKMethods)
}

func TestDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(dataSourcePrivileges, "read:pro:mac-applications") {
		t.Fatalf("dataSourcePrivileges did not render the App Installer privileges:\n%s", dataSourcePrivileges)
	}
}

// --- Plural data source (datasource_plural.go) ---

func TestPluralDataSourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, pluralDataSourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("pluralDataSourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

func TestPluralDataSourceSDKMethods_MatchReadCalls(t *testing.T) {
	assertMatch(t, "datasource_plural.go", pluralDataSourceSDKMethods)
}

func TestPluralDataSourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(pluralDataSourcePrivileges, "read:pro:mac-applications") {
		t.Fatalf("pluralDataSourcePrivileges did not render the App Installer privileges:\n%s", pluralDataSourcePrivileges)
	}
}

// --- List resource (list_resource.go) ---

func TestListResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(pro.Privileges, listResourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("listResourceSDKMethods not present in pro.Privileges (SDK drift): %v", missing)
	}
}

func TestListResourceSDKMethods_MatchListCalls(t *testing.T) {
	assertMatch(t, "list_resource.go", listResourceSDKMethods)
}

func TestListResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(listResourcePrivileges, "read:pro:mac-applications") {
		t.Fatalf("listResourcePrivileges did not render the App Installer privileges:\n%s", listResourcePrivileges)
	}
}
