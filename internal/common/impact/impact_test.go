// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// stubSource is an in-memory Source, so the counting and wording logic is tested
// without any HTTP.
type stubSource struct {
	groups []Group
	totals Totals
	err    error

	mu    sync.Mutex
	calls int
}

func (s *stubSource) Groups(context.Context) ([]Group, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.err != nil {
		return nil, s.err
	}
	return s.groups, nil
}

func (s *stubSource) Totals(context.Context) (Totals, error) {
	if s.err != nil {
		return Totals{}, s.err
	}
	return s.totals, nil
}

func (s *stubSource) groupCalls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func testSource() *stubSource {
	return &stubSource{
		totals: Totals{ManagedComputers: 300, ManagedMobileDevices: 60},
		groups: []Group{
			{PlatformID: "uuid-all", JamfProID: "1", Name: "All Managed Clients", DeviceType: DeviceTypeComputer, Smart: true, MembershipCount: 200},
			{PlatformID: "uuid-mkt", JamfProID: "12", Name: "Marketing", DeviceType: DeviceTypeComputer, Smart: true, MembershipCount: 30},
			{PlatformID: "uuid-lab", JamfProID: "13", Name: "Lab Macs", DeviceType: DeviceTypeComputer, Smart: false, MembershipCount: 5},
			{PlatformID: "uuid-ipads", JamfProID: "66", Name: "All Managed iPads", DeviceType: DeviceTypeMobile, Smart: true, MembershipCount: 40},
			// Deliberate id collision across estates. Jamf Pro numbers computer
			// groups and mobile device groups independently, so id 1 exists in
			// both; this pair is what a real tenant looks like.
			{PlatformID: "uuid-ipads-1", JamfProID: "1", Name: "All Managed iPads (dup id)", DeviceType: DeviceTypeMobile, Smart: true, MembershipCount: 12},
		},
	}
}

// strSet builds a known set of strings.
func strSet(vals ...string) types.Set {
	out, diags := types.SetValueFrom(context.Background(), types.StringType, vals)
	if diags.HasError() {
		panic(diags)
	}
	return out
}

// setWithUnknownElement builds a set holding one known and one unknown element,
// which is how Terraform represents a collection referencing an object the same
// plan creates.
func setWithUnknownElement(known string) types.Set {
	out, diags := types.SetValue(types.StringType, []attr.Value{
		types.StringValue(known),
		types.StringUnknown(),
	})
	if diags.HasError() {
		panic(diags)
	}
	return out
}

func TestCacheLoadsOnceAndIndexesBothIdentifiers(t *testing.T) {
	src := testSource()
	c := NewCache(src)
	ctx := context.Background()

	byPro, ok, err := c.GroupByJamfProID(ctx, DeviceTypeComputer, "12")
	if err != nil || !ok {
		t.Fatalf("lookup by Jamf Pro id: ok=%v err=%v", ok, err)
	}
	byUUID, ok, err := c.GroupByPlatformID(ctx, "uuid-mkt")
	if err != nil || !ok {
		t.Fatalf("lookup by Platform id: ok=%v err=%v", ok, err)
	}
	if byPro.Name != "Marketing" || byUUID.Name != "Marketing" {
		t.Fatalf("both identifiers must reach the same group, got %q and %q", byPro.Name, byUUID.Name)
	}
	if _, err := c.DeviceTotals(ctx); err != nil {
		t.Fatalf("totals: %v", err)
	}
	if got := src.groupCalls(); got != 1 {
		t.Fatalf("group list must be read once per cache, read %d times", got)
	}
}

func TestCacheLoadIsSingleFlightUnderConcurrency(t *testing.T) {
	src := testSource()
	c := NewCache(src)
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			if _, _, err := c.GroupByJamfProID(context.Background(), DeviceTypeComputer, "1"); err != nil {
				t.Errorf("lookup: %v", err)
			}
		})
	}
	wg.Wait()
	if got := src.groupCalls(); got != 1 {
		t.Fatalf("concurrent callers must share one read, read %d times", got)
	}
}

func TestCacheMemoisesFailureAndLatchesNoticeOnce(t *testing.T) {
	src := &stubSource{err: errors.New("tenant unreachable")}
	c := NewCache(src)
	ctx := context.Background()

	for range 3 {
		if _, _, err := c.GroupByJamfProID(ctx, DeviceTypeComputer, "1"); err == nil {
			t.Fatal("expected the load failure to surface")
		}
	}
	if got := src.groupCalls(); got != 1 {
		t.Fatalf("a failed load must not be retried per lookup, read %d times", got)
	}
	if !c.noticeOnce() {
		t.Fatal("first notice must fire")
	}
	if c.noticeOnce() {
		t.Fatal("notice must fire only once per plan")
	}
}

func TestNilCacheIsDisabled(t *testing.T) {
	var c *Cache
	if c.Enabled() {
		t.Fatal("a nil cache must report as disabled")
	}
	if diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Planned: Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"1"}},
	}); len(diags) != 0 {
		t.Fatalf("a disabled cache must produce no diagnostics, got %d", len(diags))
	}
}

func TestResolveSingleGroupIsExact(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 30 || res.Total != 300 {
		t.Fatalf("got count=%d total=%d, want 30/300", res.Count, res.Total)
	}
	if res.Bound != BoundExact {
		t.Fatalf("a single countable source must be exact, got bound %v", res.Bound)
	}
	if pct, ok := res.Percent(); !ok || pct != 10 {
		t.Fatalf("got percent=%d ok=%v, want 10", pct, ok)
	}
}

func TestResolveMultipleSourcesBecomeUpperBound(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12", "13"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 35 {
		t.Fatalf("got count=%d, want 35", res.Count)
	}
	if res.Bound != BoundAtMost {
		t.Fatalf("overlapping groups must yield an upper bound, got %v", res.Bound)
	}
}

func TestResolveClampsSummedOverlapToTheEstate(t *testing.T) {
	// Summing overlapping group counts can exceed the estate, which would read as
	// "up to 235 of 200 computers (117%)". The estate is a hard ceiling on the
	// true figure, so the count is clamped to it. Caught on a live tenant where
	// two overlapping groups summed past the managed computer total.
	src := testSource()
	src.totals = Totals{ManagedComputers: 210}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"1", "12", "13"}, // 200 + 30 + 5 = 235
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 210 {
		t.Fatalf("got count=%d, want the count clamped to the estate size 210", res.Count)
	}
	if res.Bound != BoundAtMost {
		t.Fatalf("a clamped figure is an upper bound, got %v", res.Bound)
	}
	if pct, ok := res.Percent(); !ok || pct != 100 {
		t.Fatalf("got percent=%d ok=%v, want 100 rather than an impossible figure", pct, ok)
	}
	// The per-group numbers must still be visible, so the clamp is explicable.
	if line := strings.Join(breakdown(res), "; "); !strings.Contains(line, "All Managed Clients (200)") {
		t.Fatalf("the breakdown must still show what was counted, got %q", line)
	}
}

func TestResolveLimitationNarrowsNotBroadens(t *testing.T) {
	// The central correctness point: a limitation filters the audience, so an
	// uncountable limitation means the figure is too high, never too low.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12"},
		Unresolvable: []Unresolvable{
			{Path: "limitations.network_segment_ids", Reason: ReasonNetworkSegment, Effect: Narrows, Values: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bound != BoundAtMost {
		t.Fatalf("a narrowing input must yield an upper bound, got %v", res.Bound)
	}
	if got := figure(res.Count, res.Bound, res.DeviceType.Noun()); got != "up to 30 computers" {
		t.Fatalf("got %q, want an upper-bound phrasing", got)
	}
}

func TestResolveBroadeningTargetYieldsLowerBound(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12"},
		Unresolvable: []Unresolvable{
			{Path: "targets.user_group_ids", Reason: ReasonUserTarget, Effect: Broadens, Values: 2},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bound != BoundAtLeast {
		t.Fatalf("a broadening target must yield a lower bound, got %v", res.Bound)
	}
	if got := figure(res.Count, res.Bound, res.DeviceType.Noun()); got != "30 or more computers" {
		t.Fatalf("got %q, want a lower-bound phrasing", got)
	}
}

func TestResolveOpposingInputsAreUnbounded(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12"},
		Unresolvable: []Unresolvable{
			{Path: "limitations.ibeacon_ids", Reason: ReasonIbeacon, Effect: Narrows, Values: 1},
			{Path: "targets.user_ids", Reason: ReasonUserTarget, Effect: Broadens, Values: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bound != BoundUnknown {
		t.Fatalf("inputs pulling both ways must bound from neither side, got %v", res.Bound)
	}
	if got := figure(res.Count, res.Bound, res.DeviceType.Noun()); !strings.Contains(got, "estimated") {
		t.Fatalf("got %q, want an estimate phrasing", got)
	}
}

func TestAmbiguousInputIsUnboundedOnItsOwn(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:       DeviceTypeAny,
		PlatformGroupIDs: []string{"uuid-mkt"},
		Unresolvable: []Unresolvable{
			{Path: "activation_conditions", Reason: "expression", Effect: Ambiguous, Values: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Bound != BoundUnknown {
		t.Fatalf("an ambiguous input must bound from neither side, got %v", res.Bound)
	}
}

func TestResolveAllFlagUsesTenantTotal(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{DeviceType: DeviceTypeComputer, All: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 300 || res.Bound != BoundExact {
		t.Fatalf("got count=%d bound=%v, want 300/exact", res.Count, res.Bound)
	}
}

func TestResolveAnyDeviceTypeSpansBothEstates(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:       DeviceTypeAny,
		PlatformGroupIDs: []string{"uuid-mkt", "uuid-ipads"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 70 {
		t.Fatalf("got count=%d, want 70 across both estates", res.Count)
	}
	if res.Total != 360 {
		t.Fatalf("got total=%d, want the whole managed estate", res.Total)
	}
	if len(res.MissingGroupIDs) != 0 {
		t.Fatalf("a mixed scope must accept both group kinds, got missing %v", res.MissingGroupIDs)
	}
}

func TestResolveRejectsWrongDeviceTypeGroup(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"66"}, // a mobile device group
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 0 {
		t.Fatalf("a mobile group must not count toward a computer scope, got %d", res.Count)
	}
	if len(res.MissingGroupIDs) != 1 {
		t.Fatalf("the unusable reference must be reported, got %v", res.MissingGroupIDs)
	}
	if res.Bound != BoundAtLeast {
		t.Fatalf("an uncounted reference can only mean the figure is short, got %v", res.Bound)
	}
}

// TestGroupIDsCollideAcrossEstates pins the wire fact that made this necessary:
// Jamf Pro numbers computer groups and mobile device groups independently, so the
// same numeric id identifies two different groups. Keying the lookup on the id
// alone loses one of them, and a computer-scoped policy then reports its group as
// missing. Verified against a live tenant, where id 1 is both "All Managed
// Clients" and "All Managed iPads".
func TestGroupIDsCollideAcrossEstates(t *testing.T) {
	c := NewCache(testSource())
	ctx := context.Background()

	comp, found, err := c.GroupByJamfProID(ctx, DeviceTypeComputer, "1")
	if err != nil || !found {
		t.Fatalf("computer group 1 must resolve: found=%v err=%v", found, err)
	}
	if comp.Name != "All Managed Clients" || comp.MembershipCount != 200 {
		t.Fatalf("computer group 1 resolved to the wrong group: %+v", comp)
	}

	mob, found, err := c.GroupByJamfProID(ctx, DeviceTypeMobile, "1")
	if err != nil || !found {
		t.Fatalf("mobile group 1 must resolve: found=%v err=%v", found, err)
	}
	if mob.MembershipCount != 12 {
		t.Fatalf("mobile group 1 resolved to the wrong group: %+v", mob)
	}
}

func TestResolveCountsTheRightEstateForACollidingID(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 200 {
		t.Fatalf("got count=%d, want the computer group's 200, not the mobile group's 12", res.Count)
	}
	if len(res.MissingGroupIDs) != 0 {
		t.Fatalf("a colliding id must not read as missing, got %v", res.MissingGroupIDs)
	}
}

func TestResolveNumericGroupIDsAreAmbiguousForAMixedScope(t *testing.T) {
	// A scope spanning both estates cannot use numeric ids, because they do not
	// identify a group on their own.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeAny,
		JamfProGroupIDs: []string{"1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 0 {
		t.Fatalf("an ambiguous id must not be counted, got %d", res.Count)
	}
	if res.Bound != BoundUnknown {
		t.Fatalf("an ambiguous id bounds the figure from neither side, got %v", res.Bound)
	}
}

func TestResolveDeduplicatesRepeatedGroupReference(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:       DeviceTypeComputer,
		JamfProGroupIDs:  []string{"12"},
		PlatformGroupIDs: []string{"uuid-mkt"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 30 {
		t.Fatalf("the same group named by both identifiers must count once, got %d", res.Count)
	}
}

func TestResolvePendingReferenceIsNotDeterminable(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:      DeviceTypeComputer,
		JamfProGroupIDs: []string{"12"},
		PendingPaths:    []string{"targets.computer_group_ids"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Determinable {
		t.Fatal("a scope referencing something this plan creates must not be determinable")
	}
	if res.Count != 0 {
		t.Fatalf("an undeterminable scope must carry no count, got %d", res.Count)
	}
	if _, ok := res.Percent(); ok {
		t.Fatal("an undeterminable scope must not offer a percentage")
	}
}

func TestResolveMentionedGroupsAreNamedNotCounted(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:           DeviceTypeAny,
		PlatformGroupIDs:     []string{"uuid-mkt"},
		MentionedPlatformIDs: []string{"uuid-all"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 30 {
		t.Fatalf("a mentioned group must not be counted, got %d", res.Count)
	}
	if len(res.Mentioned) != 1 || res.Mentioned[0].Name != "All Managed Clients" {
		t.Fatalf("the mentioned group must be resolved to its name, got %v", res.Mentioned)
	}
	if lines := strings.Join(caveats(res), "\n"); !strings.Contains(lines, "All Managed Clients (200)") {
		t.Fatalf("the caveat must name the group and its size, got %q", lines)
	}
}

func TestDeltaSplitsAdditionsAndRemovals(t *testing.T) {
	prior := Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12", "13"}}
	planned := Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12", "1"}}
	added, removed := Delta(prior, planned)
	if len(added.JamfProGroupIDs) != 1 || added.JamfProGroupIDs[0] != "1" {
		t.Fatalf("added groups wrong: %v", added.JamfProGroupIDs)
	}
	if len(removed.JamfProGroupIDs) != 1 || removed.JamfProGroupIDs[0] != "13" {
		t.Fatalf("removed groups wrong: %v", removed.JamfProGroupIDs)
	}
}

func TestReportSkipsUnchangedScope(t *testing.T) {
	c := NewCache(testSource())
	s := Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12"}}
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   s,
		Planned: s,
	})
	if len(diags) != 0 {
		t.Fatalf("an unchanged scope must not alert, got %q", diags[0].Detail())
	}
}

func TestReportSkipsUnchangedScopeWhenOrderDiffers(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12", "13"}},
		Planned: Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"13", "12"}},
	})
	if len(diags) != 0 {
		t.Fatalf("scope collections are unordered, so a reordering must not alert: %q", diags[0].Detail())
	}
}

func TestReportUpdateStatesAdditionsAndRemovals(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"13"}},
		Planned: Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12"}},
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "adding 30 computers") {
		t.Fatalf("detail must state what is being added: %q", detail)
	}
	if !strings.Contains(detail, "removing 5 computers") {
		t.Fatalf("detail must state what is being removed: %q", detail)
	}
	if !strings.Contains(detail, snapshotNote) {
		t.Fatalf("every alert must carry the snapshot caveat: %q", detail)
	}
}

func TestReportCreateAndDeleteWording(t *testing.T) {
	c := NewCache(testSource())
	s := Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12"}}

	create := Report(context.Background(), Request{
		Cache: c, Path: path.Root("scope"), Label: "policy",
		Action: ActionCreate, Planned: s,
	})
	if len(create) != 1 || !strings.Contains(create[0].Summary(), "will be scoped to 30 of 300 computers") {
		t.Fatalf("create wording wrong: %+v", create)
	}

	del := Report(context.Background(), Request{
		Cache: c, Path: path.Root("scope"), Label: "policy",
		Action: ActionDelete, Prior: s,
	})
	if len(del) != 1 || !strings.Contains(del[0].Summary(), "removing this policy affects") {
		t.Fatalf("delete wording wrong: %+v", del)
	}
	if !strings.Contains(del[0].Detail(), "will stop receiving this policy") {
		t.Fatalf("delete detail must say what stops: %q", del[0].Detail())
	}
}

func TestReportPendingReferenceExplainsWhyNoFigure(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:  c,
		Path:   path.Root("scope"),
		Label:  "policy",
		Action: ActionCreate,
		Planned: Scope{
			DeviceType:   DeviceTypeComputer,
			PendingPaths: []string{"targets.computer_group_ids"},
		},
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if !strings.Contains(diags[0].Summary(), "cannot be determined during plan") {
		t.Fatalf("summary must say the figure is unavailable: %q", diags[0].Summary())
	}
	detail := diags[0].Detail()
	for _, want := range []string{"targets.computer_group_ids", "no membership until it has been applied"} {
		if !strings.Contains(detail, want) {
			t.Fatalf("detail missing %q: %q", want, detail)
		}
	}
}

func TestReportEmptyScopeIsSilent(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache: c, Path: path.Root("scope"), Label: "policy",
		Action: ActionCreate, Planned: Scope{DeviceType: DeviceTypeComputer},
	})
	if len(diags) != 0 {
		t.Fatalf("a scope naming nothing must not alert, got %d", len(diags))
	}
}

func TestReportUnavailableTenantWarnsOnceAndNeverErrors(t *testing.T) {
	c := NewCache(&stubSource{err: errors.New("tenant unreachable")})
	s := Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12"}}

	first := Report(context.Background(), Request{
		Cache: c, Path: path.Root("scope"), Label: "policy",
		Action: ActionCreate, Planned: s,
	})
	if len(first) != 1 {
		t.Fatalf("expected one notice, got %d", len(first))
	}
	if first[0].Severity().String() != "Warning" {
		t.Fatalf("an unreadable tenant must never fail a plan, got severity %s", first[0].Severity())
	}
	second := Report(context.Background(), Request{
		Cache: c, Path: path.Root("scope"), Label: "policy",
		Action: ActionCreate, Planned: s,
	})
	if len(second) != 0 {
		t.Fatalf("the notice must not repeat per resource, got %d", len(second))
	}
}

func TestReportScopeableKindNamesTheKnockOn(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache: c, Path: path.Root("criteria"), Kind: Scopeable,
		Label: "smart computer group", Action: ActionUpdate,
		Prior:   Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"13"}},
		Planned: Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: []string{"12"}},
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if !strings.Contains(diags[0].Detail(), "changes what every object scoped to it applies to") {
		t.Fatalf("a scopeable alert must name the knock-on effect: %q", diags[0].Detail())
	}
}

func TestBreakdownSummarisesBeyondTheListedGroups(t *testing.T) {
	src := testSource()
	for i := range 10 {
		src.groups = append(src.groups, Group{
			PlatformID: "extra-" + string(rune('a'+i)), JamfProID: "90" + string(rune('0'+i)),
			Name: "Extra " + string(rune('A'+i)), DeviceType: DeviceTypeComputer, MembershipCount: int64(i + 1),
		})
	}
	c := NewCache(src)
	ids := []string{"12"}
	for i := range 10 {
		ids = append(ids, "90"+string(rune('0'+i)))
	}
	res, err := Resolve(context.Background(), c, Scope{DeviceType: DeviceTypeComputer, JamfProGroupIDs: ids})
	if err != nil {
		t.Fatal(err)
	}
	line := strings.Join(breakdown(res), "; ")
	if !strings.Contains(line, "and 6 more") {
		t.Fatalf("a long group list must be summarised, got %q", line)
	}
}

func TestScopeBuilderTreatsUnknownCollectionAsPending(t *testing.T) {
	b := NewScopeBuilder(context.Background(), DeviceTypeComputer)
	b.JamfProGroups("targets.computer_group_ids", types.SetUnknown(types.StringType))
	s := b.Scope()
	if len(s.PendingPaths) != 1 || s.PendingPaths[0] != "targets.computer_group_ids" {
		t.Fatalf("an unknown collection must be recorded as pending, got %v", s.PendingPaths)
	}
}

func TestScopeBuilderTreatsUnknownElementAsPendingAndKeepsTheKnownOne(t *testing.T) {
	// The common shape: a policy scoped to one existing group plus one the same
	// plan creates. The known id is still worth reading, but the figure cannot be
	// completed, so the path is recorded as pending.
	b := NewScopeBuilder(context.Background(), DeviceTypeComputer)
	b.JamfProGroups("targets.computer_group_ids", setWithUnknownElement("12"))
	s := b.Scope()
	if len(s.JamfProGroupIDs) != 1 || s.JamfProGroupIDs[0] != "12" {
		t.Fatalf("the known id must still be read, got %v", s.JamfProGroupIDs)
	}
	if len(s.PendingPaths) != 1 {
		t.Fatalf("the unknown element must mark the path pending, got %v", s.PendingPaths)
	}
}

func TestScopeBuilderReadsKnownValuesAndSkipsNullCollections(t *testing.T) {
	ctx := context.Background()
	b := NewScopeBuilder(ctx, DeviceTypeComputer)
	b.JamfProGroups("targets.computer_group_ids", strSet("12", "13")).
		Devices("targets.computer_ids", types.SetNull(types.StringType)).
		Narrows("limitations.network_segment_ids", strSet("7"), ReasonNetworkSegment)
	s := b.Scope()
	if len(s.JamfProGroupIDs) != 2 {
		t.Fatalf("known ids must be read, got %v", s.JamfProGroupIDs)
	}
	if len(s.DeviceIDs) != 0 || len(s.PendingPaths) != 0 {
		t.Fatalf("a null collection means absent, not pending: %v / %v", s.DeviceIDs, s.PendingPaths)
	}
	if len(s.Unresolvable) != 1 || s.Unresolvable[0].Effect != Narrows {
		t.Fatalf("a limitation must be recorded as narrowing, got %+v", s.Unresolvable)
	}
}

func TestScopeBuilderIgnoresEmptyUnresolvableCollections(t *testing.T) {
	b := NewScopeBuilder(context.Background(), DeviceTypeComputer)
	b.Narrows("limitations.ibeacon_ids", types.SetNull(types.StringType), ReasonIbeacon)
	if got := b.Scope(); len(got.Unresolvable) != 0 {
		t.Fatalf("an absent limitation must not be reported, got %+v", got.Unresolvable)
	}
}
