// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package impact

import (
	"context"
	"errors"
	"fmt"
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
	// members maps a group's Platform id to its device management ids. A group
	// absent from this map has unreadable membership, which is how the fallback to
	// approximate counting is exercised.
	members map[string][]string
	// memberErr forces every membership read to fail.
	memberErr error
	// computersByFilter maps an inventory filter to the management ids it matches.
	computersByFilter map[string][]string
	// filterErr forces every inventory read to fail.
	filterErr error

	mu          sync.Mutex
	calls       int
	memberCalls map[string]int
	filterCalls map[string]int
}

func (s *stubSource) Members(_ context.Context, platformID string) ([]string, error) {
	s.mu.Lock()
	if s.memberCalls == nil {
		s.memberCalls = map[string]int{}
	}
	s.memberCalls[platformID]++
	s.mu.Unlock()
	if s.memberErr != nil {
		return nil, s.memberErr
	}
	ids, ok := s.members[platformID]
	if !ok {
		return nil, errors.New("membership unavailable")
	}
	return ids, nil
}

// computersByFilter maps an inventory filter to the management ids it matches.
// A filter absent from this map resolves to no devices, which is what a real
// tenant returns for a building nobody is assigned to.
func (s *stubSource) ComputerManagementIDs(_ context.Context, filter string) ([]string, error) {
	s.mu.Lock()
	if s.filterCalls == nil {
		s.filterCalls = map[string]int{}
	}
	s.filterCalls[filter]++
	s.mu.Unlock()
	if s.filterErr != nil {
		return nil, s.filterErr
	}
	return s.computersByFilter[filter], nil
}

func (s *stubSource) filterCallCount(filter string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.filterCalls[filter]
}

func (s *stubSource) memberCallCount(platformID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memberCalls[platformID]
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
		// Membership expressed in device management ids, deliberately overlapping:
		// Marketing and Lab Macs share d-30, and both sit inside All Managed Clients.
		members: map[string][]string{
			"uuid-all": deviceIDs(1, 200),
			"uuid-mkt": deviceIDs(1, 30),
			"uuid-lab": {"d-30", "d-31", "d-32", "d-33", "d-34"},
			// A mobile group's members are drawn from a disjoint range: a device
			// belongs to one estate only, so the two sides never share an identifier.
			"uuid-ipads": deviceIDs(1000, 40),
		},
	}
}

// deviceIDs builds a contiguous run of synthetic management ids.
func deviceIDs(from, count int) []string {
	out := make([]string, 0, count)
	for i := range count {
		out = append(out, fmt.Sprintf("d-%d", from+i))
	}
	return out
}

// computerRefs builds computer-estate group references from numeric ids.
func computerRefs(ids ...string) []ProGroupRef {
	out := make([]ProGroupRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, ProGroupRef{DeviceType: DeviceTypeComputer, ID: id})
	}
	return out
}

// mobileRefs builds mobile-estate group references from numeric ids.
func mobileRefs(ids ...string) []ProGroupRef {
	out := make([]ProGroupRef, 0, len(ids))
	for _, id := range ids {
		out = append(out, ProGroupRef{DeviceType: DeviceTypeMobile, ID: id})
	}
	return out
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
		Planned: Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("1")},
	}); len(diags) != 0 {
		t.Fatalf("a disabled cache must produce no diagnostics, got %d", len(diags))
	}
}

func TestResolveSingleGroupIsExact(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
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

func TestResolveOverlappingGroupsAreDeduplicated(t *testing.T) {
	// Marketing holds d-1..d-30, Lab Macs holds d-30..d-34; they share d-30. The
	// summed counts would say 35, which is the number no administrator wants: it
	// implies five more computers are affected than actually are.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12", "13"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 34 {
		t.Fatalf("got count=%d, want 34 distinct devices (35 summed, sharing one)", res.Count)
	}
	if !res.Exact {
		t.Fatal("membership was readable, so the figure must come from set arithmetic")
	}
	if res.Bound != BoundExact {
		t.Fatalf("a deduplicated union needs no upper-bound hedge, got %v", res.Bound)
	}
}

func TestResolveFallsBackToSummedCountsWhenMembershipUnreadable(t *testing.T) {
	// Membership reads failing must degrade the figure, not lose it.
	src := testSource()
	src.memberErr = errors.New("no privilege to read group membership")
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12", "13"),
	})
	if err != nil {
		t.Fatalf("a membership failure must not fail resolution: %v", err)
	}
	if res.Count != 35 {
		t.Fatalf("got count=%d, want the summed 35 on the fallback path", res.Count)
	}
	if res.Exact {
		t.Fatal("the fallback figure must not claim to be exact")
	}
	if res.Bound != BoundAtMost {
		t.Fatalf("summed counts may double-count, so the figure is an upper bound, got %v", res.Bound)
	}
}

func TestResolveFallsBackWhenMembershipDisagreesWithTheCount(t *testing.T) {
	// The membership list and the membership count come from different services.
	// If they disagree the set is not trustworthy enough for exact arithmetic.
	src := testSource()
	src.members["uuid-mkt"] = []string{"d-1", "d-2"} // count says 30
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Exact {
		t.Fatal("a membership set that disagrees with the count must not be used")
	}
	if res.Count != 30 {
		t.Fatalf("got count=%d, want the group's own count of 30", res.Count)
	}
}

func TestResolveClampsSummedOverlapToTheEstate(t *testing.T) {
	// On the fallback path, summing overlapping group counts can exceed the estate,
	// which would read as "up to 235 of 210 computers (111%)". The estate is a hard
	// ceiling on the true figure, so the count is clamped to it. Caught on a live
	// tenant where two overlapping groups summed past the managed computer total.
	src := testSource()
	src.totals = Totals{ManagedComputers: 210}
	src.memberErr = errors.New("membership unavailable") // force the fallback path
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("1", "12", "13"), // 200 + 30 + 5 = 235
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
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
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
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
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
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
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
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("66"), // a mobile device group
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
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("1"),
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

func TestResolveMixedEstateScopeResolvesBothSidesCorrectly(t *testing.T) {
	// The reason group references carry their estate: id 1 exists in both, so a
	// resource targeting computer groups and mobile device groups side by side —
	// an ebook does — must resolve each against the right estate. Computer group 1
	// holds 200, mobile group 1 holds 12; a scope naming both must reach 212, not
	// 400 and not 24.
	c := NewCache(testSource())
	refs := append(computerRefs("1"), mobileRefs("1")...)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeAny,
		ProGroups:  refs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 212 {
		t.Fatalf("got count=%d, want 212 — 200 computers plus 12 mobile devices", res.Count)
	}
	if len(res.MissingGroupIDs) != 0 {
		t.Fatalf("both references must resolve, got missing %v", res.MissingGroupIDs)
	}
	if res.Total != 360 {
		t.Fatalf("got total=%d, want the whole managed estate", res.Total)
	}
}

func TestResolveDeduplicatesRepeatedGroupReference(t *testing.T) {
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:       DeviceTypeComputer,
		ProGroups:        computerRefs("12"),
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
		DeviceType:   DeviceTypeComputer,
		ProGroups:    computerRefs("12"),
		PendingPaths: []string{"targets.computer_group_ids"},
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
	prior := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12", "13")}
	planned := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12", "1")}
	added, removed := Delta(prior, planned)
	if len(added.ProGroups) != 1 || added.ProGroups[0].ID != "1" {
		t.Fatalf("added groups wrong: %v", added.ProGroups)
	}
	if len(removed.ProGroups) != 1 || removed.ProGroups[0].ID != "13" {
		t.Fatalf("removed groups wrong: %v", removed.ProGroups)
	}
}

func TestReportSkipsAnUnchangedResource(t *testing.T) {
	// Terraform calls a plan modifier for every resource in the configuration, so
	// an object with no diff must stay silent or every plan would carry one alert
	// per scoped object.
	c := NewCache(testSource())
	s := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")}
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   s,
		Planned: s,
		Changed: false,
	})
	if len(diags) != 0 {
		t.Fatalf("an untouched object must not alert, got %q", diags[0].Detail())
	}
}

func TestReportAlertsWhenOnlyThePayloadChanges(t *testing.T) {
	// Adding a script to a policy alters what every computer in its scope receives.
	// The audience has not moved, but the change still reaches those devices — and
	// the resource diff shows what changed without ever saying how many devices it
	// reaches.
	c := NewCache(testSource())
	s := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")}
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   s,
		Planned: s,
		Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if s := diags[0].Summary(); !strings.Contains(s, "affects 30 of 300 computers") {
		t.Fatalf("summary must still give the audience: %q", s)
	}
	d := diags[0].Detail()
	if !strings.Contains(d, "The scope is unchanged; these computers will receive the updated policy.") {
		t.Fatalf("detail must explain why the alert is here: %q", d)
	}
	if strings.Contains(d, "adding") || strings.Contains(d, "removing") {
		t.Fatalf("nothing entered or left scope, so no delta may be claimed: %q", d)
	}
}

func TestReportClaimsNoDeltaWhenScopeOnlyReorders(t *testing.T) {
	// Scope collections are unordered, so a reordering moves nobody in or out — the
	// alert may report the audience but must not invent a delta.
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Prior:   Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12", "13")},
		Planned: Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("13", "12")},
		Changed: true,
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if d := diags[0].Detail(); strings.Contains(d, "adding") || strings.Contains(d, "removing") {
		t.Fatalf("a reordering moves nobody: %q", d)
	}
}

func TestReportUpdateStatesAdditionsAndRemovals(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Changed: true,
		Prior:   Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("13")},
		Planned: Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")},
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
	s := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")}

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
	s := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")}

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
		Label: "smart computer group", Action: ActionUpdate, Changed: true,
		Prior:   Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("13")},
		Planned: Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")},
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
	res, err := Resolve(context.Background(), c, Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs(ids...)})
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
	b.ProGroups("targets.computer_group_ids", DeviceTypeComputer, types.SetUnknown(types.StringType))
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
	b.ProGroups("targets.computer_group_ids", DeviceTypeComputer, setWithUnknownElement("12"))
	s := b.Scope()
	if len(s.ProGroups) != 1 || s.ProGroups[0].ID != "12" {
		t.Fatalf("the known id must still be read, got %v", s.ProGroups)
	}
	if len(s.PendingPaths) != 1 {
		t.Fatalf("the unknown element must mark the path pending, got %v", s.PendingPaths)
	}
}

func TestScopeBuilderReadsKnownValuesAndSkipsNullCollections(t *testing.T) {
	ctx := context.Background()
	b := NewScopeBuilder(ctx, DeviceTypeComputer)
	b.ProGroups("targets.computer_group_ids", DeviceTypeComputer, strSet("12", "13")).
		Devices("targets.computer_ids", types.SetNull(types.StringType)).
		Narrows("limitations.network_segment_ids", strSet("7"), ReasonNetworkSegment)
	s := b.Scope()
	if len(s.ProGroups) != 2 {
		t.Fatalf("known ids must be read, got %v", s.ProGroups)
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

func TestResolveSubtractsExcludedGroupMembershipExactly(t *testing.T) {
	// The gap exact arithmetic closes: an exclusion previously narrowed the figure
	// by an unstated amount. Marketing holds d-1..d-30; Lab Macs holds d-30..d-34,
	// of which only d-30 is in Marketing. Excluding Lab Macs must remove exactly
	// that one device.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:        DeviceTypeComputer,
		ProGroups:         computerRefs("12"),
		ExcludedProGroups: computerRefs("13"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 29 {
		t.Fatalf("got count=%d, want 30 less the one shared device", res.Count)
	}
	if !res.Exact || res.Bound != BoundExact {
		t.Fatalf("an exactly subtracted exclusion needs no hedge, got exact=%v bound=%v", res.Exact, res.Bound)
	}
	for _, u := range res.Unresolvable {
		if u.Path == "excluded groups" {
			t.Fatal("a subtracted exclusion must not also be reported as an unquantified caveat")
		}
	}
}

func TestResolveExcludedGroupBecomesCaveatOnTheFallbackPath(t *testing.T) {
	src := testSource()
	src.memberErr = errors.New("membership unavailable")
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:        DeviceTypeComputer,
		ProGroups:         computerRefs("12"),
		ExcludedProGroups: computerRefs("13"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 30 {
		t.Fatalf("got count=%d, want the unsubtracted 30", res.Count)
	}
	if res.Bound != BoundAtMost {
		t.Fatalf("an unsubtracted exclusion makes the figure an upper bound, got %v", res.Bound)
	}
	var named bool
	for _, u := range res.Unresolvable {
		if u.Path == "excluded groups" && strings.Contains(u.Reason, "Lab Macs") {
			named = true
		}
	}
	if !named {
		t.Fatalf("the caveat must name the group whose membership could not be read, got %+v", res.Unresolvable)
	}
}

func TestResolveAllComputersMinusExcludedGroup(t *testing.T) {
	// A tenant-wide target with an exclusion is the common "everything except"
	// shape, and it resolves exactly: the estate less the excluded membership.
	src := testSource()
	src.totals = Totals{ManagedComputers: 200}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:        DeviceTypeComputer,
		All:               true,
		ExcludedProGroups: computerRefs("12"), // 30 members
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 170 {
		t.Fatalf("got count=%d, want 200 less the 30 excluded", res.Count)
	}
	if !res.Exact || res.Bound != BoundExact {
		t.Fatalf("got exact=%v bound=%v, want an unhedged figure", res.Exact, res.Bound)
	}
}

func TestResolveNamedComputersJoinTheUnionWithoutDoubleCounting(t *testing.T) {
	// The point of resolving named computers to management identifiers: computer 5
	// is already inside Marketing, so naming it individually must not add a second
	// device to the figure. Previously the two were separate additive terms and the
	// result carried an upper-bound hedge it no longer needs.
	src := testSource()
	src.computersByFilter = map[string][]string{
		"id==5 or id==6": {"d-1", "d-500"}, // d-1 is inside Marketing, d-500 is not
	}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"), // Marketing, d-1..d-30
		DeviceIDs:  []string{"5", "6"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 31 {
		t.Fatalf("got count=%d, want 31 — Marketing's 30 plus the one named computer outside it", res.Count)
	}
	if !res.Exact || res.Bound != BoundExact {
		t.Fatalf("got exact=%v bound=%v, want an unhedged figure", res.Exact, res.Bound)
	}
}

func TestResolveBuildingTargetIsCountedExactly(t *testing.T) {
	src := testSource()
	src.computersByFilter = map[string][]string{
		"userAndLocation.buildingId==321": {"d-900", "d-901"},
	}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:  DeviceTypeComputer,
		BuildingIDs: []string{"321"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 2 {
		t.Fatalf("got count=%d, want the 2 computers in that building", res.Count)
	}
	if res.Bound != BoundExact {
		t.Fatalf("a counted building needs no hedge, got %v", res.Bound)
	}
	for _, u := range res.Unresolvable {
		if u.Path == "building_ids" || u.Path == "targets.building_ids" {
			t.Fatalf("a counted building must not also be reported as a caveat: %+v", u)
		}
	}
}

func TestResolveExcludedBuildingIsSubtractedExactly(t *testing.T) {
	src := testSource()
	src.computersByFilter = map[string][]string{
		"userAndLocation.buildingId==321": {"d-1", "d-2"}, // both inside Marketing
	}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:          DeviceTypeComputer,
		ProGroups:           computerRefs("12"), // 30 members, d-1..d-30
		ExcludedBuildingIDs: []string{"321"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 28 {
		t.Fatalf("got count=%d, want 30 less the 2 in the excluded building", res.Count)
	}
	if res.Bound != BoundExact {
		t.Fatalf("a subtracted building needs no hedge, got %v", res.Bound)
	}
}

func TestResolveMobileBuildingStaysUnresolved(t *testing.T) {
	// Mobile devices filter buildings by name while a scope block carries ids, so
	// the mobile estate keeps the unresolved treatment — and it must broaden, since
	// a target can only add devices.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:  DeviceTypeMobile,
		ProGroups:   mobileRefs("66"),
		BuildingIDs: []string{"321"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, u := range res.Unresolvable {
		if u.Path == "building_ids" {
			found = true
			if u.Effect != Broadens {
				t.Fatalf("an unresolvable building target broadens, got %v", u.Effect)
			}
		}
	}
	if !found {
		t.Fatalf("the unresolvable building must be reported: %+v", res.Unresolvable)
	}
	if res.Bound != BoundAtLeast {
		t.Fatalf("got bound=%v, want a lower bound", res.Bound)
	}
}

func TestResolveInventoryFilterIsReadOncePerFilter(t *testing.T) {
	src := testSource()
	src.computersByFilter = map[string][]string{
		"userAndLocation.buildingId==321": {"d-900"},
	}
	c := NewCache(src)
	s := Scope{DeviceType: DeviceTypeComputer, BuildingIDs: []string{"321"}}
	for range 5 {
		if _, err := Resolve(context.Background(), c, s); err != nil {
			t.Fatal(err)
		}
	}
	if got := src.filterCallCount("userAndLocation.buildingId==321"); got != 1 {
		t.Fatalf("a filter must be read once per plan, read %d times", got)
	}
}

func TestResolveAbandonedExactAttemptDoesNotLeaveCaveatsBehind(t *testing.T) {
	// The exact strategy resolves named categories before it knows whether it can
	// finish. If it then gives up, anything it recorded must not be reported
	// alongside the approximate figure that replaces it.
	src := testSource()
	src.memberErr = errors.New("membership unavailable") // force the fallback
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType:  DeviceTypeMobile,
		ProGroups:   mobileRefs("66"),
		BuildingIDs: []string{"321"},
	})
	if err != nil {
		t.Fatal(err)
	}
	var seen int
	for _, u := range res.Unresolvable {
		if u.Path == "building_ids" || u.Path == "targets.building_ids" {
			seen++
		}
	}
	if seen != 1 {
		t.Fatalf("the building must be reported exactly once, got %d: %+v", seen, res.Unresolvable)
	}
}

func TestResolveDeviceOnlyScopeResolvesWithoutReadingAnyGroupMembership(t *testing.T) {
	// A scope naming only individual computers costs one inventory read and no
	// group membership reads at all.
	src := testSource()
	src.computersByFilter = map[string][]string{
		"id==5 or id==6 or id==7": {"d-700", "d-701", "d-702"},
	}
	c := NewCache(src)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		DeviceIDs:  []string{"5", "6", "7"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Count != 3 || res.Bound != BoundExact {
		t.Fatalf("got count=%d bound=%v, want an exact 3", res.Count, res.Bound)
	}
	if len(src.memberCalls) != 0 {
		t.Fatalf("a scope naming no groups must read no group membership, got %v", src.memberCalls)
	}
}

func TestMembersReadOncePerGroupAcrossResolvesAndConcurrency(t *testing.T) {
	// Membership is the per-plan cost, so it must be read once per group however
	// many resources reference it.
	src := testSource()
	c := NewCache(src)
	s := Scope{DeviceType: DeviceTypeComputer, ProGroups: computerRefs("12")}

	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			if _, err := Resolve(context.Background(), c, s); err != nil {
				t.Errorf("resolve: %v", err)
			}
		})
	}
	wg.Wait()
	if got := src.memberCallCount("uuid-mkt"); got != 1 {
		t.Fatalf("membership of a group must be read once per plan, read %d times", got)
	}
}

func TestReportLiftedExclusionCountsAsAnAddition(t *testing.T) {
	// Removing an exclusion puts devices back in scope, so it belongs on the
	// "adding" side of the delta even though it is written under exclusions.
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Changed: true,
		Prior: Scope{
			DeviceType:        DeviceTypeComputer,
			ProGroups:         computerRefs("12"),
			ExcludedProGroups: computerRefs("13"),
		},
		Planned: Scope{
			DeviceType: DeviceTypeComputer,
			ProGroups:  computerRefs("12"),
		},
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	detail := diags[0].Detail()
	if !strings.Contains(detail, "adding") {
		t.Fatalf("lifting an exclusion adds devices: %q", detail)
	}
	if strings.Contains(detail, "removing") {
		t.Fatalf("nothing is leaving scope here: %q", detail)
	}
}

func TestReportNewExclusionCountsAsARemoval(t *testing.T) {
	c := NewCache(testSource())
	diags := Report(context.Background(), Request{
		Cache:   c,
		Path:    path.Root("scope"),
		Label:   "policy",
		Action:  ActionUpdate,
		Changed: true,
		Prior: Scope{
			DeviceType: DeviceTypeComputer,
			ProGroups:  computerRefs("12"),
		},
		Planned: Scope{
			DeviceType:        DeviceTypeComputer,
			ProGroups:         computerRefs("12"),
			ExcludedProGroups: computerRefs("13"),
		},
	})
	if len(diags) != 1 {
		t.Fatalf("expected one alert, got %d", len(diags))
	}
	if d := diags[0].Detail(); !strings.Contains(d, "removing") {
		t.Fatalf("adding an exclusion removes devices: %q", d)
	}
}

func TestResolveSplitsFigureAcrossEstates(t *testing.T) {
	// A merged "3 of 5 devices" hides the distinction that matters most: three Macs
	// and three iPads are not the same change. A scope spanning both estates must
	// report each side with its own denominator.
	c := NewCache(testSource())
	refs := append(computerRefs("12"), mobileRefs("66")...)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeAny,
		ProGroups:  refs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.PerEstate[DeviceTypeComputer]; got != 30 {
		t.Fatalf("computer side = %d, want 30", got)
	}
	if got := res.PerEstate[DeviceTypeMobile]; got != 40 {
		t.Fatalf("mobile side = %d, want 40", got)
	}
	if !res.Exact {
		t.Fatal("both estates' membership was readable, so the figure must be exact")
	}
	want := "30 of 300 computers and 40 of 60 mobile devices"
	if got := summarise(res); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestResolveSingleEstateScopeIsNotSplit(t *testing.T) {
	// A scope confined to one estate already says which in its noun, so a split
	// would only add noise.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeComputer,
		ProGroups:  computerRefs("12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.PerEstate != nil {
		t.Fatalf("a single-estate scope must not be split, got %v", res.PerEstate)
	}
	if got := summarise(res); got != "30 of 300 computers (10%)" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveMixedEstateScopeNamingOneEstateUsesThatEstatesDenominator(t *testing.T) {
	// A resource that can span both estates but names only computers must not be
	// measured against a denominator that includes mobile devices it never touches.
	c := NewCache(testSource())
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeAny,
		ProGroups:  computerRefs("12"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := summarise(res); got != "30 of 300 computers" {
		t.Fatalf("got %q, want the computer estate's own denominator", got)
	}
}

func TestResolveNamedEstateWithNoMembersStillReportsZero(t *testing.T) {
	// Caught live: an ebook naming a mobile device group that currently holds
	// nothing reported only the computer side, silently hiding that the mobile side
	// was targeted at all. The estate is named by the scope, so it must appear.
	src := testSource()
	src.groups = append(src.groups, Group{
		PlatformID: "uuid-empty-mobile", JamfProID: "99", Name: "Empty iPads",
		DeviceType: DeviceTypeMobile, Smart: true, MembershipCount: 0,
	})
	src.members["uuid-empty-mobile"] = nil
	c := NewCache(src)
	refs := append(computerRefs("12"), mobileRefs("99")...)
	res, err := Resolve(context.Background(), c, Scope{DeviceType: DeviceTypeAny, ProGroups: refs})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.PerEstate[DeviceTypeMobile]; !ok {
		t.Fatalf("the named mobile estate must appear even at zero, got %v", res.PerEstate)
	}
	want := "30 of 300 computers and 0 of 60 mobile devices"
	if got := summarise(res); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSplitFigureCarriesTheBoundAsALeadingPhrase(t *testing.T) {
	// "30 or more computers and 40 of 60 mobile devices" would attach the
	// qualifier to one side only, so a split moves it to the front.
	c := NewCache(testSource())
	refs := append(computerRefs("12"), mobileRefs("66")...)
	res, err := Resolve(context.Background(), c, Scope{
		DeviceType: DeviceTypeAny,
		ProGroups:  refs,
		Unresolvable: []Unresolvable{
			{Path: "targets.user_group_ids", Reason: ReasonUserTarget, Effect: Broadens, Values: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := summarise(res)
	if !strings.HasPrefix(got, "at least ") {
		t.Fatalf("got %q, want a leading qualifier", got)
	}
	if strings.Contains(got, "or more") {
		t.Fatalf("the trailing form must not be used for a split: %q", got)
	}
}
