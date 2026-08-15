// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// stubPolicySource is an in-memory PolicySource, so the sweep, the reverse index
// and the union arithmetic are all tested without HTTP.
type stubPolicySource struct {
	ids      []string
	policies map[string]*proclassic.Policy
	// listErr fails the initial listing, which is the one fatal failure.
	listErr error
	// policyErrs fails individual policy reads, which must be survivable.
	policyErrs map[string]error
	// delay holds each read briefly. Without it the stub returns so fast that
	// workers never overlap and maxConcurrent stays at 1, which would make the
	// concurrency assertion vacuous rather than wrong.
	delay time.Duration

	mu        sync.Mutex
	listCalls int
	getCalls  map[string]int
	// maxConcurrent records the high-water mark of simultaneous reads, so the
	// sweep's concurrency bound can be asserted rather than assumed.
	inFlight      int
	maxConcurrent int
}

func (s *stubPolicySource) PolicyIDs(_ context.Context) ([]string, error) {
	s.mu.Lock()
	s.listCalls++
	s.mu.Unlock()
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.ids, nil
}

func (s *stubPolicySource) Policy(_ context.Context, id string) (*proclassic.Policy, error) {
	s.mu.Lock()
	if s.getCalls == nil {
		s.getCalls = map[string]int{}
	}
	s.getCalls[id]++
	s.inFlight++
	if s.inFlight > s.maxConcurrent {
		s.maxConcurrent = s.inFlight
	}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.inFlight--
		s.mu.Unlock()
	}()
	if s.delay > 0 {
		time.Sleep(s.delay)
	}

	if err, ok := s.policyErrs[id]; ok {
		return nil, err
	}
	return s.policies[id], nil
}

// --- builders, so each test states only what it cares about ---
//
// The wire types are pointer-heavy, so the fixtures lean on new(expr) to take
// the address of a literal inline.

type policyOpt func(*proclassic.Policy)

func testPolicy(id int, name string, opts ...policyOpt) *proclassic.Policy {
	p := &proclassic.Policy{
		General: &proclassic.PolicyGeneral{
			ID: new(id), Name: new(name), Enabled: new(true),
		},
	}
	for _, o := range opts {
		o(p)
	}
	return p
}

func disabled() policyOpt {
	return func(p *proclassic.Policy) { p.General.Enabled = new(false) }
}

func withScripts(ids ...int) policyOpt {
	return func(p *proclassic.Policy) {
		items := make([]proclassic.PolicyScriptsScriptItem, 0, len(ids))
		for _, id := range ids {
			items = append(items, proclassic.PolicyScriptsScriptItem{ID: new(id)})
		}
		p.Scripts = &proclassic.PolicyScripts{Script: &items}
	}
}

func withPackages(ids ...int) policyOpt {
	return func(p *proclassic.Policy) {
		items := make([]proclassic.PolicyPackageConfigurationPackagesPackageItem, 0, len(ids))
		for _, id := range ids {
			items = append(items, proclassic.PolicyPackageConfigurationPackagesPackageItem{ID: new(id)})
		}
		p.PackageConfiguration = &proclassic.PolicyPackageConfiguration{
			Packages: &proclassic.PolicyPackageConfigurationPackages{Package: &items},
		}
	}
}

func withDiskEncryption(applyID, remediateID *int) policyOpt {
	return func(p *proclassic.Policy) {
		p.DiskEncryption = &proclassic.PolicyDiskEncryption{
			DiskEncryptionConfigurationID:          applyID,
			RemediateDiskEncryptionConfigurationID: remediateID,
		}
	}
}

// withGroupScope scopes the policy to computer groups by numeric id.
func withGroupScope(groupIDs ...int) policyOpt {
	return func(p *proclassic.Policy) {
		items := make([]proclassic.IDName, 0, len(groupIDs))
		for _, id := range groupIDs {
			items = append(items, proclassic.IDName{ID: new(id)})
		}
		p.Scope = &proclassic.PolicyScope{
			ComputerGroups: &proclassic.PolicyScopeComputerGroups{ComputerGroup: &items},
		}
	}
}

func withAllComputers() policyOpt {
	return func(p *proclassic.Policy) {
		p.Scope = &proclassic.PolicyScope{AllComputers: new(true)}
	}
}

// depCache builds a Cache wired to both stub sources.
func depCache(groups *stubSource, policies *stubPolicySource) *Cache {
	return NewCacheWithPolicies(groups, policies)
}

// twoGroupTenant is a tenant with two overlapping computer groups, which is what
// makes the union arithmetic observable: group 1 and group 2 share device "b".
func twoGroupTenant() *stubSource {
	return &stubSource{
		groups: []Group{
			{PlatformID: "u1", JamfProID: "1", Name: "Group One", DeviceType: DeviceTypeComputer, MembershipCount: 2},
			{PlatformID: "u2", JamfProID: "2", Name: "Group Two", DeviceType: DeviceTypeComputer, MembershipCount: 2},
		},
		totals:  Totals{ManagedComputers: 4},
		members: map[string][]string{"u1": {"a", "b"}, "u2": {"b", "c"}},
	}
}

func TestPolicyUses_IndexesEveryDependencyKind(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{
		ids: []string{"10", "11"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Runs script", withScripts(500), withPackages(80)),
			"11": testPolicy(11, "Encrypts", withDiskEncryption(new(59), new(60))),
		},
	}
	c := depCache(twoGroupTenant(), src)

	for _, tc := range []struct {
		kind DependencyKind
		id   string
		want string
	}{
		{DependencyScript, "500", "Runs script"},
		{DependencyPackage, "80", "Runs script"},
		{DependencyDiskEncryptionConfiguration, "59", "Encrypts"},
		// The remediate field makes the policy depend on that configuration too, so
		// editing configuration 60 must also be reported.
		{DependencyDiskEncryptionConfiguration, "60", "Encrypts"},
	} {
		uses, swept, err := c.PolicyUses(context.Background(), tc.kind, tc.id)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.kind, tc.id, err)
		}
		if swept != 2 {
			t.Errorf("%s %s: swept = %d, want 2", tc.kind, tc.id, swept)
		}
		if len(uses) != 1 || uses[0].Name != tc.want {
			t.Errorf("%s %s: uses = %+v, want one use named %q", tc.kind, tc.id, uses, tc.want)
		}
	}
}

func TestPolicyUses_SweepsOnceAcrossKindsAndConcurrentCallers(t *testing.T) {
	t.Parallel()
	ids := make([]string, 0, 30)
	policies := map[string]*proclassic.Policy{}
	for i := 1; i <= 30; i++ {
		id := string(rune('0'+i/10)) + string(rune('0'+i%10))
		ids = append(ids, id)
		policies[id] = testPolicy(i, "Policy "+id, withScripts(500))
	}
	src := &stubPolicySource{ids: ids, policies: policies, delay: 2 * time.Millisecond}
	c := depCache(twoGroupTenant(), src)

	// Terraform evaluates resources concurrently, so several dependency resources
	// can ask at once; they must share one sweep rather than each starting theirs.
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			if _, _, err := c.PolicyUses(context.Background(), DependencyScript, "500"); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	// And a different kind reuses the same sweep.
	if _, _, err := c.PolicyUses(context.Background(), DependencyPackage, "80"); err != nil {
		t.Fatal(err)
	}

	if src.listCalls != 1 {
		t.Errorf("listCalls = %d, want 1", src.listCalls)
	}
	for id, n := range src.getCalls {
		if n != 1 {
			t.Errorf("policy %s read %d times, want 1", id, n)
		}
	}
	if src.maxConcurrent > dependencySweepConcurrency {
		t.Errorf("maxConcurrent = %d, exceeds the %d bound Jamf's guidance asks for",
			src.maxConcurrent, dependencySweepConcurrency)
	}
	if src.maxConcurrent < 2 {
		t.Errorf("maxConcurrent = %d, sweep did not run concurrently at all", src.maxConcurrent)
	}
}

func TestPolicyUses_SurvivesUnreadablePolicyButNotUnreadableListing(t *testing.T) {
	t.Parallel()

	// One bad policy costs that policy's contribution, not the whole alert.
	src := &stubPolicySource{
		ids: []string{"10", "11"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Good", withScripts(500)),
		},
		policyErrs: map[string]error{"11": errors.New("boom")},
	}
	c := depCache(twoGroupTenant(), src)
	uses, swept, err := c.PolicyUses(context.Background(), DependencyScript, "500")
	if err != nil {
		t.Fatalf("one unreadable policy must not fail the sweep: %v", err)
	}
	if len(uses) != 1 {
		t.Errorf("uses = %d, want 1", len(uses))
	}
	// swept still reports what was searched, so the diagnostic does not understate it.
	if swept != 2 {
		t.Errorf("swept = %d, want 2", swept)
	}

	// An unreadable listing is fatal: an empty index would read as "nothing uses this".
	bad := depCache(twoGroupTenant(), &stubPolicySource{listErr: errors.New("nope")})
	if _, _, err := bad.PolicyUses(context.Background(), DependencyScript, "500"); err == nil {
		t.Error("unreadable listing must surface an error, not an empty index")
	}
}

func TestPolicyUses_MemoisesFailure(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{listErr: errors.New("nope")}
	c := depCache(twoGroupTenant(), src)
	for range 3 {
		if _, _, err := c.PolicyUses(context.Background(), DependencyScript, "500"); err == nil {
			t.Fatal("want error")
		}
	}
	if src.listCalls != 1 {
		t.Errorf("listCalls = %d, want 1 — a failed sweep must not be retried per resource", src.listCalls)
	}
}

func TestPolicyUses_DisabledCacheAndEmptyIDReportNothing(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{ids: []string{"10"},
		policies: map[string]*proclassic.Policy{"10": testPolicy(10, "P", withScripts(500))}}

	// A cache without a policy source is the state of a provider whose impact
	// alerts are on but which was built without dependency support.
	noPolicies := NewCache(twoGroupTenant())
	if uses, _, err := noPolicies.PolicyUses(context.Background(), DependencyScript, "500"); err != nil || uses != nil {
		t.Errorf("cache without policy source: uses=%v err=%v, want nil/nil", uses, err)
	}
	var nilCache *Cache
	if uses, _, err := nilCache.PolicyUses(context.Background(), DependencyScript, "500"); err != nil || uses != nil {
		t.Errorf("nil cache: uses=%v err=%v, want nil/nil", uses, err)
	}
	// An empty id must not trigger the sweep at all.
	c := depCache(twoGroupTenant(), src)
	if _, _, err := c.PolicyUses(context.Background(), DependencyScript, ""); err != nil {
		t.Fatal(err)
	}
	if src.listCalls != 0 {
		t.Errorf("listCalls = %d, want 0 — an empty id must not sweep", src.listCalls)
	}
}

func TestReportDependency_UnionsOverlappingScopesExactlyOnce(t *testing.T) {
	t.Parallel()
	// Two policies use script 500. Group One is {a,b}, Group Two is {b,c}, so the
	// combined audience is {a,b,c} = 3, not 2+2 = 4.
	src := &stubPolicySource{
		ids: []string{"10", "11"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Alpha", withScripts(500), withGroupScope(1)),
			"11": testPolicy(11, "Beta", withScripts(500), withGroupScope(2)),
		},
	}
	c := depCache(twoGroupTenant(), src)

	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionUpdate, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1: %v", len(diags), diags)
	}
	summary, detail := diags[0].Summary(), diags[0].Detail()
	if !strings.Contains(summary, "3 of 4 computers") {
		t.Errorf("summary must report the deduplicated union of 3, got %q", summary)
	}
	if !strings.Contains(summary, "2 policies") {
		t.Errorf("summary must name how many policies carry the change, got %q", summary)
	}
	for _, want := range []string{"Alpha", "Beta", "counted once"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail missing %q:\n%s", want, detail)
		}
	}
	if !strings.Contains(detail, "Searched 2 policies") {
		t.Errorf("detail must say what was searched:\n%s", detail)
	}
}

func TestReportDependency_DisabledPoliciesReportedSeparately(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{
		ids: []string{"10", "11"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Live", withScripts(500), withGroupScope(1)),
			"11": testPolicy(11, "Staged", disabled(), withScripts(500), withAllComputers()),
		},
	}
	c := depCache(twoGroupTenant(), src)
	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionUpdate, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1", len(diags))
	}
	summary, detail := diags[0].Summary(), diags[0].Detail()
	// The disabled policy scopes all computers; if it were counted the figure
	// would be 4 rather than the 2 the enabled policy reaches.
	if !strings.Contains(summary, "2 of 4 computers") {
		t.Errorf("a disabled policy must not contribute devices, got %q", summary)
	}
	if !strings.Contains(summary, "1 policy") {
		t.Errorf("summary should count only enabled policies, got %q", summary)
	}
	if !strings.Contains(detail, "Staged") || !strings.Contains(detail, "disabled") {
		t.Errorf("detail must still disclose the disabled policy:\n%s", detail)
	}
}

func TestReportDependency_OnlyDisabledPolicies(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{
		ids: []string{"10"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Staged", disabled(), withScripts(500), withGroupScope(1)),
		},
	}
	c := depCache(twoGroupTenant(), src)
	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionUpdate, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1", len(diags))
	}
	if !strings.Contains(diags[0].Summary(), "only by disabled policies") {
		t.Errorf("summary = %q", diags[0].Summary())
	}
	if !strings.Contains(diags[0].Detail(), "reaches no computers yet") {
		t.Errorf("detail = %q", diags[0].Detail())
	}
}

func TestReportDependency_UnusedDependency(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{
		ids:      []string{"10"},
		policies: map[string]*proclassic.Policy{"10": testPolicy(10, "Other", withScripts(999))},
	}
	c := depCache(twoGroupTenant(), src)
	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionUpdate, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1", len(diags))
	}
	// Worth reporting rather than staying silent: "nothing uses this" is the
	// reassurance an administrator editing a script wants.
	if !strings.Contains(diags[0].Summary(), "no policy uses this script") {
		t.Errorf("summary = %q", diags[0].Summary())
	}
}

func TestReportDependency_SkipsCreateUnchangedAndDisabled(t *testing.T) {
	t.Parallel()
	newSrc := func() *stubPolicySource {
		return &stubPolicySource{
			ids:      []string{"10"},
			policies: map[string]*proclassic.Policy{"10": testPolicy(10, "P", withScripts(500), withGroupScope(1))},
		}
	}

	for _, tc := range []struct {
		name string
		req  DependencyRequest
	}{
		// Nothing can reference an id the tenant has not issued yet, so a create
		// must not even sweep.
		{"create", DependencyRequest{Kind: DependencyScript, ID: "", Action: ActionCreate, Changed: true}},
		{"create with id", DependencyRequest{Kind: DependencyScript, ID: "500", Action: ActionCreate, Changed: true}},
		{"unchanged update", DependencyRequest{Kind: DependencyScript, ID: "500", Action: ActionUpdate, Changed: false}},
		{"no id", DependencyRequest{Kind: DependencyScript, ID: "", Action: ActionUpdate, Changed: true}},
	} {
		src := newSrc()
		req := tc.req
		req.Cache = depCache(twoGroupTenant(), src)
		req.Path = path.Empty()
		if diags := ReportDependency(context.Background(), req); len(diags) != 0 {
			t.Errorf("%s: diags = %v, want none", tc.name, diags)
		}
		if src.listCalls != 0 {
			t.Errorf("%s: swept the tenant when it should not have", tc.name)
		}
	}
}

func TestReportDependency_DeleteWording(t *testing.T) {
	t.Parallel()
	src := &stubPolicySource{
		ids:      []string{"10"},
		policies: map[string]*proclassic.Policy{"10": testPolicy(10, "Alpha", withScripts(500), withGroupScope(1))},
	}
	c := depCache(twoGroupTenant(), src)
	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionDelete, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1", len(diags))
	}
	if !strings.Contains(diags[0].Summary(), "removing this script affects") {
		t.Errorf("summary = %q", diags[0].Summary())
	}
	// The detail carries no delete-specific lead sentence — the headline states it —
	// so the body just has to name the affected policy.
	if !strings.Contains(diags[0].Detail(), "Used by 1 enabled policy: Alpha") {
		t.Errorf("detail = %q", diags[0].Detail())
	}
}

func TestReportDependency_SweepFailureNotifiesOnce(t *testing.T) {
	t.Parallel()
	c := depCache(twoGroupTenant(), &stubPolicySource{listErr: errors.New("tenant unreachable")})
	req := DependencyRequest{
		Cache: c, Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "s", Action: ActionUpdate, Changed: true,
	}
	first := ReportDependency(context.Background(), req)
	if len(first) != 1 || !strings.Contains(first[0].Summary(), "Impact alert unavailable") {
		t.Fatalf("first = %v, want one unavailable notice", first)
	}
	// Advisory, and once per plan: a second dependency resource stays quiet.
	if second := ReportDependency(context.Background(), req); len(second) != 0 {
		t.Errorf("second = %v, want none", second)
	}
}

func TestPolicyWireScope_ClassifiesEverySection(t *testing.T) {
	t.Parallel()

	t.Run("nil scope is empty", func(t *testing.T) {
		t.Parallel()
		got := policyWireScope(nil)
		if !got.Empty() || got.DeviceType != DeviceTypeComputer {
			t.Errorf("got %+v, want an empty computer scope", got)
		}
	})

	t.Run("targets counted, limitations narrow, user targets broaden", func(t *testing.T) {
		t.Parallel()
		s := &proclassic.PolicyScope{
			AllJssUsers: new(true),
			Computers: &proclassic.PolicyScopeComputers{
				Computer: &[]proclassic.PolicyScopeComputersComputerItem{{ID: new(7)}},
			},
			ComputerGroups: &proclassic.PolicyScopeComputerGroups{
				ComputerGroup: &[]proclassic.IDName{{ID: new(1)}},
			},
			Buildings: &proclassic.PolicyScopeBuildings{
				Building: &[]proclassic.IDName{{ID: new(3)}},
			},
			Departments: &proclassic.PolicyScopeDepartments{
				Department: &[]proclassic.IDName{{ID: new(4)}},
			},
			Limitations: &proclassic.PolicyScopeLimitations{
				NetworkSegments: &proclassic.PolicyScopeLimitationsNetworkSegments{
					NetworkSegment: &[]proclassic.IDName{{ID: new(9)}},
				},
			},
			Exclusions: &proclassic.PolicyScopeExclusions{
				ComputerGroups: &proclassic.PolicyScopeExclusionsComputerGroups{
					ComputerGroup: &[]proclassic.IDName{{ID: new(2)}},
				},
			},
		}
		got := policyWireScope(s)

		if len(got.DeviceIDs) != 1 || got.DeviceIDs[0] != "7" {
			t.Errorf("DeviceIDs = %v, want [7]", got.DeviceIDs)
		}
		if len(got.ProGroups) != 1 || got.ProGroups[0].ID != "1" ||
			got.ProGroups[0].DeviceType != DeviceTypeComputer {
			t.Errorf("ProGroups = %+v, want computer group 1", got.ProGroups)
		}
		if len(got.BuildingIDs) != 1 || len(got.DepartmentIDs) != 1 {
			t.Errorf("buildings/departments = %v/%v, want one each", got.BuildingIDs, got.DepartmentIDs)
		}
		if len(got.ExcludedProGroups) != 1 || got.ExcludedProGroups[0].ID != "2" {
			t.Errorf("ExcludedProGroups = %+v, want computer group 2", got.ExcludedProGroups)
		}

		var narrows, broadens int
		for _, u := range got.Unresolvable {
			switch u.Effect {
			case Narrows:
				narrows++
			case Broadens:
				broadens++
			}
		}
		// A network segment limitation narrows; all_jss_users broadens. Getting the
		// direction wrong would tell the reader the figure errs the wrong way.
		if narrows != 1 {
			t.Errorf("narrowing inputs = %d, want 1 (the network segment limitation)", narrows)
		}
		if broadens != 1 {
			t.Errorf("broadening inputs = %d, want 1 (all_jss_users)", broadens)
		}
	})
}

func TestCombineScopes_DropsPerPolicyExclusionsAsNarrowing(t *testing.T) {
	t.Parallel()
	// An exclusion belongs to the policy that declares it. A computer excluded
	// from policy A but targeted by policy B still receives the dependency, so
	// subtracting A's exclusion from the combined audience would undercount.
	a := Scope{
		DeviceType:        DeviceTypeComputer,
		ProGroups:         []ProGroupRef{{DeviceTypeComputer, "1"}},
		ExcludedDeviceIDs: []string{"x"},
	}
	b := Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  []ProGroupRef{{DeviceTypeComputer, "2"}},
	}

	// One contributing policy keeps its exclusions, since they apply to the whole
	// (single-policy) audience.
	if only := combineScopes([]Scope{a}); len(only.ExcludedDeviceIDs) != 1 {
		t.Errorf("a single policy must keep its exclusions, got %+v", only.ExcludedDeviceIDs)
	}

	got := combineScopes([]Scope{a, b})
	if len(got.ExcludedDeviceIDs) != 0 {
		t.Errorf("ExcludedDeviceIDs = %v, want dropped once two policies combine", got.ExcludedDeviceIDs)
	}
	if len(got.ProGroups) != 2 {
		t.Errorf("ProGroups = %+v, want both groups unioned", got.ProGroups)
	}
	var found bool
	for _, u := range got.Unresolvable {
		if u.Effect == Narrows && strings.Contains(u.Reason, "per policy") {
			found = true
		}
	}
	if !found {
		t.Errorf("dropping exclusions must be disclosed as narrowing, got %+v", got.Unresolvable)
	}
}

// TestDependencyAlert_IsMachineReadable pins the wording against the regexes
// documented in the CI section of docs/guides/impact-alerts.md.
//
// Alerts are consumed from `terraform plan -json` and matched by regex, so the
// phrasing is a contract, not prose. Compacting the detail is fine; moving a count
// out of its anchor phrase silently breaks every pipeline gating on it.
func TestDependencyAlert_IsMachineReadable(t *testing.T) {
	t.Parallel()

	// The shared figure pattern from the guide, which must match a dependency
	// headline exactly as it matches a policy or profile one.
	figure := regexp.MustCompile(`(scoped to|affects) (up to |at least |an estimated )?([0-9]+)( of ([0-9]+))?`)
	// Dependency-specific anchors.
	via := regexp.MustCompile(`via ([0-9]+) (policy|policies)$`)
	usedBy := regexp.MustCompile(`Used by ([0-9]+) enabled (policy|policies):`)
	alsoDisabled := regexp.MustCompile(`Also ([0-9]+) disabled (policy|policies), delivering nothing until enabled:`)
	searched := regexp.MustCompile(`Searched ([0-9]+) (policy|policies)\.`)

	src := &stubPolicySource{
		ids: []string{"10", "11", "12"},
		policies: map[string]*proclassic.Policy{
			"10": testPolicy(10, "Alpha", withScripts(500), withGroupScope(1)),
			"11": testPolicy(11, "Beta", withScripts(500), withGroupScope(2)),
			"12": testPolicy(12, "Staged", disabled(), withScripts(500), withGroupScope(1)),
		},
	}

	for _, action := range []Action{ActionUpdate, ActionDelete} {
		c := depCache(twoGroupTenant(), src)
		diags := ReportDependency(context.Background(), DependencyRequest{
			Cache: c, Path: path.Empty(), Kind: DependencyScript,
			ID: "500", Name: "my script", Action: action, Changed: true,
		})
		if len(diags) != 1 {
			t.Fatalf("action %v: diags = %d, want 1", action, len(diags))
		}
		summary, detail := diags[0].Summary(), diags[0].Detail()

		m := figure.FindStringSubmatch(summary)
		if m == nil {
			t.Fatalf("action %v: summary does not match the documented figure regex: %q", action, summary)
		}
		// Group 3 is the count, group 5 the denominator: 3 of 4 computers.
		if m[3] != "3" || m[5] != "4" {
			t.Errorf("action %v: parsed count/total = %q/%q, want 3/4 from %q", action, m[3], m[5], summary)
		}
		if vm := via.FindStringSubmatch(summary); vm == nil || vm[1] != "2" {
			t.Errorf("action %v: summary must end with the enabled-policy count: %q", action, summary)
		}

		for name, re := range map[string]*regexp.Regexp{
			"Used by":       usedBy,
			"Also disabled": alsoDisabled,
			"Searched":      searched,
		} {
			if !re.MatchString(detail) {
				t.Errorf("action %v: detail lost the %q anchor:\n%s", action, name, detail)
			}
		}
		if sm := searched.FindStringSubmatch(detail); sm != nil && sm[1] != "3" {
			t.Errorf("action %v: searched count = %q, want 3", action, sm[1])
		}
	}

	// The singular forms must stay parseable too — "1 policy", not "1 policys".
	single := &stubPolicySource{
		ids:      []string{"10"},
		policies: map[string]*proclassic.Policy{"10": testPolicy(10, "Alpha", withScripts(500), withGroupScope(1))},
	}
	diags := ReportDependency(context.Background(), DependencyRequest{
		Cache: depCache(twoGroupTenant(), single), Path: path.Empty(), Kind: DependencyScript,
		ID: "500", Name: "my script", Action: ActionUpdate, Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("diags = %d, want 1", len(diags))
	}
	if !via.MatchString(diags[0].Summary()) {
		t.Errorf("singular summary must still match the via anchor: %q", diags[0].Summary())
	}
	if !searched.MatchString(diags[0].Detail()) {
		t.Errorf("singular detail must still match the searched anchor:\n%s", diags[0].Detail())
	}
}

func TestCombineScopes_AggregatesRepeatedCaveatsIntoOneLine(t *testing.T) {
	t.Parallel()
	// Live tenants hit this hard: a script used by 56 policies, most of them with
	// exclusions and some limited by a network segment, produced one identical
	// caveat line per policy — dozens of copies of the same sentence — which
	// buried the figure the alert exists to convey.
	scopes := make([]Scope, 0, 20)
	for range 20 {
		scopes = append(scopes, Scope{
			DeviceType:        DeviceTypeComputer,
			ProGroups:         []ProGroupRef{{DeviceTypeComputer, "1"}},
			ExcludedDeviceIDs: []string{"x"},
			Unresolvable: []Unresolvable{{
				Path:   "scope.limitations.network_segments",
				Reason: ReasonNetworkSegment,
				Effect: Narrows,
				Values: 1,
			}},
		})
	}

	got := combineScopes(scopes)

	counts := map[string]int{}
	values := map[string]int{}
	for _, u := range got.Unresolvable {
		counts[u.Path]++
		values[u.Path] += u.Values
	}
	for path, n := range counts {
		if n != 1 {
			t.Errorf("%s appears %d times, want exactly 1 aggregated entry", path, n)
		}
	}
	// Aggregating must not lose the total: 20 policies each excluding one device.
	if values["policy exclusions"] != 20 {
		t.Errorf("policy exclusions total = %d, want 20", values["policy exclusions"])
	}
	if values["scope.limitations.network_segments"] != 20 {
		t.Errorf("network segment total = %d, want 20", values["scope.limitations.network_segments"])
	}

	// And the rendered caveats must contain each line once.
	res := Resolution{DeviceType: DeviceTypeComputer, Unresolvable: got.Unresolvable}
	rendered := strings.Join(caveats(res), "\n")
	if n := strings.Count(rendered, exclusionsAreNotCombinable); n != 1 {
		t.Errorf("exclusion caveat rendered %d times, want 1:\n%s", n, rendered)
	}
	if n := strings.Count(rendered, ReasonNetworkSegment); n != 1 {
		t.Errorf("network segment caveat rendered %d times, want 1:\n%s", n, rendered)
	}
}

func TestCombineScopes_DeduplicatesAndPropagatesAll(t *testing.T) {
	t.Parallel()
	a := Scope{DeviceType: DeviceTypeComputer,
		ProGroups: []ProGroupRef{{DeviceTypeComputer, "1"}}, DeviceIDs: []string{"d1"}}
	b := Scope{DeviceType: DeviceTypeComputer,
		ProGroups: []ProGroupRef{{DeviceTypeComputer, "1"}}, DeviceIDs: []string{"d1", "d2"}, All: true}

	got := combineScopes([]Scope{a, b})
	if len(got.ProGroups) != 1 {
		t.Errorf("ProGroups = %+v, want deduplicated to one", got.ProGroups)
	}
	if len(got.DeviceIDs) != 2 {
		t.Errorf("DeviceIDs = %v, want [d1 d2]", got.DeviceIDs)
	}
	if !got.All {
		t.Error("All must propagate: one policy scoping every computer makes the union tenant-wide")
	}
}
