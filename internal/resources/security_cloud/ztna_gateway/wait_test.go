// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package ztna_gateway

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/securitycloud"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// scriptedReader returns a gatewayReader that answers with the given states in
// order, repeating the last one once exhausted, and a pointer to the read count.
//
// The reads are what the assertions are really about: the wait's contract is as
// much about how many times it goes to the API as about what it concludes.
func scriptedReader(states ...string) (gatewayReader, *int) {
	reads := 0
	return func(_ context.Context, _ string) (*securitycloud.Gateway, error) {
		state := states[min(reads, len(states)-1)]
		reads++
		return &securitycloud.Gateway{Status: &securitycloud.GatewayStatus{State: state}}, nil
	}, &reads
}

// TestWaitForGatewayUp_SatisfiedOnlyByUp pins which reported states end the wait.
//
// UP ends it. PENDING does not, which is the whole point. DOWN does not either, and
// specifically does not end it as a failure: the 2026-08-31 internet-form probe never
// produced DOWN, so whether it can appear transiently mid-provisioning on that form is
// unknown, and treating it as terminal on that ignorance would fail an apply that was
// about to succeed. The same day's IPsec probe did settle at DOWN, which is why that
// form is excluded from the wait entirely rather than being made to fail fast here.
func TestWaitForGatewayUp_SatisfiedOnlyByUp(t *testing.T) {
	cases := []struct {
		name      string
		states    []string
		wantReads int
	}{
		{
			name:      "already up ends the wait on the first read",
			states:    []string{securitycloud.GatewayStatusStateUp},
			wantReads: 1,
		},
		{
			name: "pending is not terminal",
			states: []string{
				securitycloud.GatewayStatusStatePending,
				securitycloud.GatewayStatusStatePending,
				securitycloud.GatewayStatusStateUp,
			},
			wantReads: 3,
		},
		{
			name: "down is not terminal either",
			states: []string{
				securitycloud.GatewayStatusStateDown,
				securitycloud.GatewayStatusStateDown,
				securitycloud.GatewayStatusStateUp,
			},
			wantReads: 3,
		},
		{
			name: "a disabled report is not terminal",
			states: []string{
				securitycloud.GatewayStatusStateDisabled,
				securitycloud.GatewayStatusStateUp,
			},
			wantReads: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			read, reads := scriptedReader(tc.states...)
			observed, lastState, reachedUp := waitForGatewayUp(context.Background(), read, "a1b2", time.Millisecond)

			if !reachedUp {
				t.Fatalf("the wait was not satisfied; last state %q after %d reads", lastState, *reads)
			}
			if lastState != securitycloud.GatewayStatusStateUp {
				t.Errorf("last state = %q, want %q", lastState, securitycloud.GatewayStatusStateUp)
			}
			if *reads != tc.wantReads {
				t.Errorf("reads = %d, want %d", *reads, tc.wantReads)
			}
			if observed == nil {
				t.Error("the wait must hand back the representation it read, or the caller pays for a further read")
			}
		})
	}
}

// TestWaitForGatewayUp_AlreadyUpDoesNotSleep pins the property that makes the update
// path affordable.
//
// An update that does not re-provision the gateway — a name or contact change — finds
// it already UP. Waiting has to cost nothing in that case, because it is what lets
// Update wait unconditionally instead of trying to work out from the plan whether the
// change re-provisions anything. The interval here is far longer than the assertion
// window, so a single sleep would fail the test.
func TestWaitForGatewayUp_AlreadyUpDoesNotSleep(t *testing.T) {
	read, reads := scriptedReader(securitycloud.GatewayStatusStateUp)

	started := time.Now()
	_, _, reachedUp := waitForGatewayUp(context.Background(), read, "a1b2", 30*time.Second)
	elapsed := time.Since(started)

	if !reachedUp {
		t.Fatal("an already-operational gateway must satisfy the wait")
	}
	if *reads != 1 {
		t.Errorf("reads = %d, want 1", *reads)
	}
	if elapsed > time.Second {
		t.Errorf("the wait slept for %s on an already-operational gateway; it must return immediately", elapsed)
	}
}

// TestWaitForGatewayUp_BudgetExpiryReportsTheStateReached pins what the wait hands
// back when the caller's context budget runs out: not satisfied, plus the last state
// it saw, which is what the warning is worded from.
func TestWaitForGatewayUp_BudgetExpiryReportsTheStateReached(t *testing.T) {
	cases := []struct {
		name  string
		state string
	}{
		{name: "stuck provisioning", state: securitycloud.GatewayStatusStatePending},
		{name: "reported down", state: securitycloud.GatewayStatusStateDown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			read, reads := scriptedReader(tc.state)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()

			observed, lastState, reachedUp := waitForGatewayUp(ctx, read, "a1b2", time.Millisecond)

			if reachedUp {
				t.Fatal("a gateway that never reports itself operational must not satisfy the wait")
			}
			if lastState != tc.state {
				t.Errorf("last state = %q, want %q", lastState, tc.state)
			}
			if *reads < 2 {
				t.Errorf("reads = %d, want the wait to have kept polling", *reads)
			}
			if observed == nil {
				t.Error("the last successful read must survive an exhausted budget, or the caller must read again on a dead context")
			}
		})
	}
}

// TestWaitForGatewayUp_ReadFailureEndsTheWait pins that a read failure stops the wait
// instead of being swallowed, matching waitForBenchmarkSync in
// internal/resources/cbengine/benchmark.
//
// Swallowing it would turn a broken credential into a full-budget spin. Ending the
// wait costs nothing, because the caller reads the gateway again afterwards and that
// read produces the real error diagnostic.
func TestWaitForGatewayUp_ReadFailureEndsTheWait(t *testing.T) {
	reads := 0
	read := func(_ context.Context, _ string) (*securitycloud.Gateway, error) {
		reads++
		return nil, errors.New("the read failed")
	}

	observed, lastState, reachedUp := waitForGatewayUp(context.Background(), read, "a1b2", 30*time.Second)

	if reachedUp {
		t.Fatal("a read failure must not satisfy the wait")
	}
	if reads != 1 {
		t.Errorf("reads = %d, want 1 — the wait must not retry a failing read", reads)
	}
	if lastState != "" {
		t.Errorf("last state = %q, want empty — nothing was ever read", lastState)
	}
	if observed != nil {
		t.Error("nothing was read, so the wait must hand back nothing")
	}
}

// TestGatewayWaitsForUp_Gates pins both gate conditions.
//
// Both are measured, not cautious. The 2026-08-31 IPsec probe pointed a gateway's
// peer address at nothing and watched it settle at DOWN after 35 seconds and stay
// there, never reaching UP — so waiting on that form would burn the whole budget and
// then warn about a gateway behaving as designed, since building the Jamf side before
// the customer side is the normal order of work. The enabled gate is definitional: a
// disabled gateway reports DISABLED, so the wait could only ever run out.
func TestGatewayWaitsForUp_Gates(t *testing.T) {
	cases := []struct {
		name string
		plan GatewayResourceModel
		want bool
	}{
		{
			name: "enabled internet gateway waits",
			plan: GatewayResourceModel{Enabled: types.BoolValue(true)},
			want: true,
		},
		{
			name: "ipsec gateway does not wait",
			plan: GatewayResourceModel{Enabled: types.BoolValue(true), IPSec: &IPSecModel{}},
			want: false,
		},
		{
			name: "disabled internet gateway does not wait",
			plan: GatewayResourceModel{Enabled: types.BoolValue(false)},
			want: false,
		},
		{
			name: "disabled ipsec gateway does not wait",
			plan: GatewayResourceModel{Enabled: types.BoolValue(false), IPSec: &IPSecModel{}},
			want: false,
		},
		{
			name: "an unknown enabled value skips the wait rather than risking a hang",
			plan: GatewayResourceModel{Enabled: types.BoolUnknown()},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			plan := tc.plan
			if got := gatewayWaitsForUp(&plan); got != tc.want {
				t.Errorf("gatewayWaitsForUp = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestAppendGatewayWaitWarning_NamesTheStateReached pins that an exhausted wait is
// reported as a warning and never an error, and that the wording distinguishes the
// states, because they call for different responses.
//
// Never an error: by the time the wait is exhausted the gateway exists and is
// billable, and an error returned from a create the server accepted makes Terraform
// taint the resource and destroy and recreate it on the next apply. The warning-only
// assertion here is what keeps the resource in state — a diagnostic with no error in
// it cannot discard one.
func TestAppendGatewayWaitWarning_NamesTheStateReached(t *testing.T) {
	cases := []struct {
		name      string
		op        gatewayWaitOperation
		lastState string
		wantText  []string
	}{
		{
			name:      "still provisioning reads as slowness on create",
			op:        gatewayWaitCreate,
			lastState: securitycloud.GatewayStatusStatePending,
			wantText: []string{
				"created successfully",
				"taking longer",
				"stalled",
				"settles on a later refresh",
				"`create`",
			},
		},
		{
			name:      "still provisioning reads as slowness on update",
			op:        gatewayWaitUpdate,
			lastState: securitycloud.GatewayStatusStatePending,
			wantText: []string{
				"updated successfully",
				"taking longer",
				"`update`",
			},
		},
		{
			name:      "down reads as a fault to investigate, not slowness",
			op:        gatewayWaitCreate,
			lastState: securitycloud.GatewayStatusStateDown,
			wantText: []string{
				"created successfully",
				"unreachable or degraded",
				"more likely a fault than slowness",
				"admin UI",
			},
		},
		{
			name:      "any other state is repeated verbatim",
			op:        gatewayWaitCreate,
			lastState: securitycloud.GatewayStatusStateDisabled,
			wantText: []string{
				securitycloud.GatewayStatusStateDisabled,
				"not a status the provider expects",
			},
		},
		{
			name:      "an unread status says so",
			op:        gatewayWaitCreate,
			lastState: "",
			wantText: []string{
				"could not read the gateway's status",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			appendGatewayWaitWarning(&diags, tc.op, tc.lastState)

			if diags.HasError() {
				t.Fatalf("an exhausted wait must never error, or a billable gateway is tainted and replaced: %v", diags)
			}
			if diags.WarningsCount() != 1 {
				t.Fatalf("warnings = %d, want 1: %v", diags.WarningsCount(), diags)
			}
			warning := diags.Warnings()[0]
			text := warning.Summary() + " " + warning.Detail()
			for _, want := range tc.wantText {
				if !strings.Contains(text, want) {
					t.Errorf("warning %q does not mention %q", text, want)
				}
			}
			if strings.Contains(text, "dedicatedIps") || strings.Contains(text, "GET ") {
				t.Errorf("warning %q leaks wire vocabulary", text)
			}
		})
	}
}
