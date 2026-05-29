// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package providerdata

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// newFakeClient builds a non-network jamfplatform.Client suitable for tests that
// exercise ConfigurePro. The version fetcher is overridden so no HTTP call is made;
// the client itself is only needed because ConfigurePro constructs pro.New(pd.Client)
// to return to callers.
func newFakeClient() *jamfplatform.Client {
	return jamfplatform.NewClient("http://127.0.0.1:1", "test-id", "test-secret")
}

// countSeverity returns the number of diagnostics at the given severity.
func countSeverity(d diag.Diagnostics, sev diag.Severity) int {
	n := 0
	for _, e := range d {
		if e.Severity() == sev {
			n++
		}
	}
	return n
}

func TestConfigurePro_NilProviderData(t *testing.T) {
	client, diags := ConfigurePro(context.Background(), nil, "", "jamfplatform_pro_test")
	if client != nil {
		t.Errorf("expected nil client for nil providerData, got %v", client)
	}
	if diags.HasError() {
		t.Errorf("expected no diagnostics for nil providerData, got %v", diags)
	}
}

func TestConfigurePro_WrongType(t *testing.T) {
	client, diags := ConfigurePro(context.Background(), "not a Data value", "", "jamfplatform_pro_test")
	if client != nil {
		t.Errorf("expected nil client for wrong providerData type, got %v", client)
	}
	if !diags.HasError() {
		t.Fatalf("expected error diagnostic for wrong providerData type, got %v", diags)
	}
	if !strings.Contains(diags[0].Summary(), "Unexpected Configure Type") {
		t.Errorf("expected 'Unexpected Configure Type' summary, got %q", diags[0].Summary())
	}
}

func TestConfigurePro_HappyPath_NoMinVer(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "11.27.0", nil
		},
	}
	client, diags := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test")
	if client == nil {
		t.Fatal("expected non-nil client on happy path")
	}
	if diags.HasError() {
		t.Errorf("expected no errors, got %v", diags)
	}
	if countSeverity(diags, diag.SeverityWarning) != 0 {
		t.Errorf("expected no warnings at floor, got %v", diags)
	}
}

func TestConfigurePro_MinVerGate_Satisfied(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "11.27.0", nil
		},
	}
	_, diags := ConfigurePro(context.Background(), pd, "11.5.0", "jamfplatform_pro_test")
	if diags.HasError() {
		t.Errorf("expected no error when tenant >= minVer, got %v", diags)
	}
}

func TestConfigurePro_MinVerGate_Failed(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "10.0.0", nil
		},
	}
	_, diags := ConfigurePro(context.Background(), pd, "11.5.0", "jamfplatform_pro_test")
	if !diags.HasError() {
		t.Fatalf("expected error when tenant < minVer, got %v", diags)
	}
}

// TestConfigurePro_FloorWarning_EmittedOnce verifies the provider-floor advisory
// warning fires at most once per Data value even with many Pro Configure calls.
func TestConfigurePro_FloorWarning_EmittedOnce(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "10.0.0", nil
		},
	}
	_, diags1 := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test_a")
	if got := countSeverity(diags1, diag.SeverityWarning); got != 1 {
		t.Fatalf("first Configure: expected 1 warning, got %d (%v)", got, diags1)
	}

	for i := range 4 {
		_, diagsN := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test_more")
		if got := countSeverity(diagsN, diag.SeverityWarning); got != 0 {
			t.Errorf("subsequent Configure #%d: expected 0 warnings, got %d (%v)", i+2, got, diagsN)
		}
	}
}

// TestConfigurePro_FetchError_NotCached verifies a transient fetch error in the
// first Configure does not poison subsequent Configures — they retry the fetch.
func TestConfigurePro_FetchError_NotCached(t *testing.T) {
	calls := 0
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			calls++
			if calls == 1 {
				return "", errors.New("transient network error")
			}
			return "11.27.0", nil
		},
	}
	// First call: empty minVer → fetch error is swallowed, client returned.
	client, diags := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test_a")
	if client == nil {
		t.Fatal("expected client even when fetch errors with empty minVer")
	}
	if diags.HasError() {
		t.Errorf("expected no error diagnostics with empty minVer + fetch failure, got %v", diags)
	}

	// Second call: non-empty minVer → must retry the fetch and succeed.
	_, diags2 := ConfigurePro(context.Background(), pd, "11.0.0", "jamfplatform_pro_test_b")
	if diags2.HasError() {
		t.Errorf("expected retry to succeed on second Configure, got %v", diags2)
	}
	if calls < 2 {
		t.Errorf("expected versionFetcher to be retried, only called %d times", calls)
	}
}

// TestConfigurePro_FetchError_HardWithMinVer verifies a fetch failure surfaces as
// a Configure-time error when the resource declares a non-empty minVer.
func TestConfigurePro_FetchError_HardWithMinVer(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "", errors.New("503 service unavailable")
		},
	}
	client, diags := ConfigurePro(context.Background(), pd, "11.5.0", "jamfplatform_pro_test")
	if client != nil {
		t.Errorf("expected nil client when fetch fails with non-empty minVer, got %v", client)
	}
	if !diags.HasError() {
		t.Fatalf("expected error diagnostic, got %v", diags)
	}
	if !strings.Contains(diags[0].Summary(), "Failed to read Jamf Pro tenant version") {
		t.Errorf("expected 'Failed to read Jamf Pro tenant version' summary, got %q", diags[0].Summary())
	}
}

// TestConfigurePro_ShortCircuit_EmptyMinVerAfterFloorHandled verifies that once the
// provider-floor advisory has been considered for a Data value, subsequent Configure
// calls with empty minVer skip the version fetch entirely. This keeps acceptance-test
// noise low when many empty-minVer Pro resources reuse the same Data.
func TestConfigurePro_ShortCircuit_EmptyMinVerAfterFloorHandled(t *testing.T) {
	calls := 0
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			calls++
			return "11.27.0", nil
		},
	}
	// First Configure: fetches and considers floor (no warning at floor).
	_, diags := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test_a")
	if diags.HasError() {
		t.Fatalf("unexpected error on first Configure: %v", diags)
	}
	if calls != 1 {
		t.Fatalf("first Configure: expected 1 fetch, got %d", calls)
	}

	// Subsequent empty-minVer Configures must not fetch.
	for range 5 {
		_, _ = ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_test_more")
	}
	if calls != 1 {
		t.Errorf("expected 1 total fetch after 6 empty-minVer Configures, got %d", calls)
	}

	// A Configure with non-empty minVer must still consult the cached version (no new
	// fetch — cached value is reused, gate evaluated against it).
	_, diags = ConfigurePro(context.Background(), pd, "11.0.0", "jamfplatform_pro_test_gated")
	if diags.HasError() {
		t.Errorf("non-empty minVer Configure unexpectedly errored: %v", diags)
	}
	if calls != 1 {
		t.Errorf("expected fetch count unchanged after non-empty minVer Configure (cache hit), got %d", calls)
	}
}

// ProClassic Configure tests are intentionally narrower than the Pro suite:
// ConfigureProClassic and ConfigurePro share the configureSub core, so the
// version-cache, fetch-retry, fast-path-short-circuit, and gate-satisfied paths
// are already exercised through the Pro tests above. The tests below cover the
// proclassic-specific wiring (factory choice, returned type) and the
// cross-client invariants we need to keep (shared one-shot floor warning).

func TestConfigureProClassic_NilProviderData(t *testing.T) {
	client, diags := ConfigureProClassic(context.Background(), nil, "", "jamfplatform_pro_test")
	if client != nil {
		t.Errorf("expected nil client for nil providerData, got %v", client)
	}
	if diags.HasError() {
		t.Errorf("expected no diagnostics for nil providerData, got %v", diags)
	}
}

func TestConfigureProClassic_WrongType(t *testing.T) {
	client, diags := ConfigureProClassic(context.Background(), "not a Data value", "", "jamfplatform_pro_test")
	if client != nil {
		t.Errorf("expected nil client for wrong providerData type, got %v", client)
	}
	if !diags.HasError() {
		t.Fatalf("expected error diagnostic for wrong providerData type, got %v", diags)
	}
	if !strings.Contains(diags[0].Summary(), "Unexpected Configure Type") {
		t.Errorf("expected 'Unexpected Configure Type' summary, got %q", diags[0].Summary())
	}
}

func TestConfigureProClassic_HappyPath_NoMinVer(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "11.27.0", nil
		},
	}
	client, diags := ConfigureProClassic(context.Background(), pd, "", "jamfplatform_pro_test")
	if client == nil {
		t.Fatal("expected non-nil proclassic client on happy path")
	}
	if diags.HasError() {
		t.Errorf("expected no errors, got %v", diags)
	}
}

func TestConfigureProClassic_MinVerGate_Failed(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "10.0.0", nil
		},
	}
	_, diags := ConfigureProClassic(context.Background(), pd, "11.5.0", "jamfplatform_pro_test")
	if !diags.HasError() {
		t.Fatalf("expected error when tenant < minVer, got %v", diags)
	}
}

// TestConfigureProClassic_SharesFloorStateWithPro verifies that ConfigurePro and
// ConfigureProClassic share the one-shot floor-warning machinery on the same
// Data value. A config that mixes both kinds of resources must only see one
// warning regardless of order.
func TestConfigureProClassic_SharesFloorStateWithPro(t *testing.T) {
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			return "10.0.0", nil
		},
	}
	_, diags1 := ConfigurePro(context.Background(), pd, "", "jamfplatform_pro_a")
	if got := countSeverity(diags1, diag.SeverityWarning); got != 1 {
		t.Fatalf("first (Pro) Configure: expected 1 warning, got %d (%v)", got, diags1)
	}
	_, diags2 := ConfigureProClassic(context.Background(), pd, "", "jamfplatform_pro_b")
	if got := countSeverity(diags2, diag.SeverityWarning); got != 0 {
		t.Errorf("subsequent (ProClassic) Configure: expected 0 warnings, got %d (%v)", got, diags2)
	}
}

// TestFiredOnce_NamespacedLatches verifies that distinct keys each fire exactly
// once and that repeat keys are suppressed thereafter. Used by cross-API bridging
// paths to avoid duplicate warnings per provider invocation.
func TestFiredOnce_NamespacedLatches(t *testing.T) {
	pd := &Data{}
	if !pd.FiredOnce("foo") {
		t.Error("first call for key 'foo' should return true")
	}
	if pd.FiredOnce("foo") {
		t.Error("second call for key 'foo' should return false")
	}
	if !pd.FiredOnce("bar") {
		t.Error("first call for distinct key 'bar' should return true")
	}
	if pd.FiredOnce("bar") {
		t.Error("second call for key 'bar' should return false")
	}
	if pd.FiredOnce("foo") {
		t.Error("third call for key 'foo' should still return false")
	}
}

// TestGetJamfProVersion_CachesSuccess verifies successful fetches are not re-issued.
func TestGetJamfProVersion_CachesSuccess(t *testing.T) {
	calls := 0
	pd := &Data{Client: newFakeClient(),
		versionFetcher: func(_ context.Context) (string, error) {
			calls++
			return "11.27.0", nil
		},
	}
	for i := range 3 {
		v, err := pd.GetJamfProVersion(context.Background())
		if err != nil {
			t.Fatalf("call %d: unexpected error %v", i, err)
		}
		if v != "11.27.0" {
			t.Errorf("call %d: got %q, want %q", i, v, "11.27.0")
		}
	}
	if calls != 1 {
		t.Errorf("expected exactly 1 fetch for cached success, got %d", calls)
	}
}

// TestEnrollmentWriteLock_StableIdentity verifies the shared enrollment write lock
// is the same *sync.Mutex instance on every call for a given Data value, so the
// user-initiated-enrollment-settings (/v4) and re-enrollment-settings (/v1) resources
// — which receive the same *Data by pointer at Configure — serialize against one lock.
func TestEnrollmentWriteLock_StableIdentity(t *testing.T) {
	pd := New(newFakeClient())
	a := pd.EnrollmentWriteLock()
	b := pd.EnrollmentWriteLock()
	if a == nil {
		t.Fatal("EnrollmentWriteLock returned nil")
	}
	if a != b {
		t.Errorf("expected the same mutex instance across calls, got %p and %p", a, b)
	}
	// Sanity: the returned lock is usable and starts unlocked.
	if !a.TryLock() {
		t.Fatal("fresh enrollment write lock should be acquirable")
	}
	a.Unlock()

	// Distinct Data values must not share a lock.
	if other := New(newFakeClient()).EnrollmentWriteLock(); other == a {
		t.Error("distinct Data values unexpectedly share one enrollment write lock")
	}
}
