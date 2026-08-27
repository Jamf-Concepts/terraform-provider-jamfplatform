// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_grouped_gateway

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

// syntheticMethodBacking maps a synthetic SDK helper onto the generated method
// whose privileges it actually consumes. The privilege registry omits
// Resolve<X>ByName / Apply<X> by design, so a call site using one would otherwise
// look like it needs no privileges at all.
var syntheticMethodBacking = map[string]string{
	"ResolveZtnaGroupedGatewayV1ByName": "ListZtnaGroupedGatewaysV1",
}

// calledMethods returns the distinct SDK client method names invoked in the given
// source file, restricted to methods known to the SDK privilege registry.
func calledMethods(t *testing.T, filename string) map[string]bool {
	t.Helper()
	src, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	called := map[string]bool{}
	for _, m := range clientCallRe.FindAllStringSubmatch(string(src), -1) {
		name := m[1]
		if backing, ok := syntheticMethodBacking[name]; ok {
			name = backing
		}
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

func TestResourceSDKMethods_KnownToSDK(t *testing.T) {
	if missing := permissions.Missing(securitycloud.Privileges, resourceSDKMethods...); len(missing) > 0 {
		t.Fatalf("resourceSDKMethods not present in securitycloud.Privileges (SDK drift): %v", missing)
	}
}

func TestResourceSDKMethods_MatchCRUDCalls(t *testing.T) {
	assertMatch(t, "crud.go", resourceSDKMethods)
}

func TestResourcePrivileges_Rendered(t *testing.T) {
	if !strings.Contains(resourcePrivileges, "ztna:create") {
		t.Fatalf("resourcePrivileges did not render the grouped gateway privileges:\n%s", resourcePrivileges)
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
	if !strings.Contains(dataSourcePrivileges, "ztna:read") {
		t.Fatalf("dataSourcePrivileges did not render the grouped gateway privileges:\n%s", dataSourcePrivileges)
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
	if !strings.Contains(pluralDataSourcePrivileges, "ztna:read") {
		t.Fatalf("pluralDataSourcePrivileges did not render the grouped gateway privileges:\n%s", pluralDataSourcePrivileges)
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
	if !strings.Contains(listResourcePrivileges, "ztna:read") {
		t.Fatalf("listResourcePrivileges did not render the grouped gateway privileges:\n%s", listResourcePrivileges)
	}
}

// --- enum sourcing ---

// TestRoutingStrategyValuesComeFromSDK guards the single-source-of-truth claim: the
// OneOf validator and the documented list are both generated from this slice, so a
// silently empty one would take both down at once.
func TestRoutingStrategyValuesComeFromSDK(t *testing.T) {
	values := routingStrategyValues()
	if len(values) == 0 {
		t.Fatal("routing strategy value set is empty; the OneOf validator would accept nothing")
	}
	want := map[string]bool{"ACTIVE_STANDBY": false, "RANDOM": false, "NEAREST": false}
	for _, v := range values {
		if _, ok := want[v]; ok {
			want[v] = true
		}
	}
	for strategy, seen := range want {
		if !seen {
			t.Errorf("routing strategy set is missing %s", strategy)
		}
	}
}

// TestRecoveryDelayValues pins the exact durations Jamf Security Cloud accepts.
// Anything else — including the Go zero value, which is what an omitted Int64
// would send — is rejected with a message naming the whole set.
func TestRecoveryDelayValues(t *testing.T) {
	got := recoveryDelayValues()
	want := []int64{300, 1800, 3600, 10800, 28800}
	if len(got) != len(want) {
		t.Fatalf("recovery delay set = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("recovery delay set = %v, want %v", got, want)
		}
	}
	for _, v := range got {
		if v == 0 {
			t.Error("zero must not be an accepted duration")
		}
	}
}

// TestRecoveryDelayValuesAreCopied guards the slice against a caller mutating the
// package-level set for the rest of the process.
func TestRecoveryDelayValuesAreCopied(t *testing.T) {
	first := recoveryDelayValues()
	first[0] = 999
	if recoveryDelayValues()[0] == 999 {
		t.Error("recoveryDelayValues must return a fresh slice per call")
	}
}
