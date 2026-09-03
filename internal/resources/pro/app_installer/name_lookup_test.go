// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"errors"
	"testing"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// stubTitleLister serves a fixed catalog, recording the filter it was asked for.
// It deliberately IGNORES the filter, reproducing Jamf Pro's case-insensitive
// glob: every candidate comes back and the resolver must do the narrowing.
type stubTitleLister struct {
	titles []pro.AppTitle
	err    error
	filter string
	calls  int
}

func (s *stubTitleLister) ListAppInstallerTitlesV1(ctx context.Context, sort []string, filter string) ([]pro.AppTitle, error) {
	s.calls++
	s.filter = filter
	return s.titles, s.err
}

func TestResolveTitleIDByName_ExactMatch(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{{ID: "Composer", TitleName: "Jamf Composer"}}}
	id, err := resolveTitleIDByName(context.Background(), l, "Jamf Composer")
	if err != nil {
		t.Fatalf("expected the exact name to resolve, got %v", err)
	}
	if id != "Composer" {
		t.Errorf("id = %q, want Composer", id)
	}
	if l.filter != `titleName=="Jamf Composer"` {
		t.Errorf("filter = %q", l.filter)
	}
}

// Jamf Pro's titleName filter matches case-insensitively, so an off-casing name
// comes back as a candidate. Accepting it would store the user's spelling and
// then have Read rewrite it to the canonical name — a perpetual diff. The
// resolver must reject it instead.
func TestResolveTitleIDByName_RejectsCaseMismatch(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{{ID: "Composer", TitleName: "Jamf Composer"}}}
	if _, err := resolveTitleIDByName(context.Background(), l, "jamf composer"); !errors.Is(err, errTitleNotInCatalog) {
		t.Fatalf("expected errTitleNotInCatalog for an off-casing name, got %v", err)
	}
}

// The same filter also globs on `*`, so a name containing one over-matches. The
// exact check must still pick out the one title actually named that.
func TestResolveTitleIDByName_RejectsGlobOverMatch(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{
		{ID: "7F0", TitleName: "Jamf Sync"},
		{ID: "Composer", TitleName: "Jamf Composer"},
	}}
	if _, err := resolveTitleIDByName(context.Background(), l, "Jamf*"); !errors.Is(err, errTitleNotInCatalog) {
		t.Fatalf("expected errTitleNotInCatalog for a glob that matches no exact name, got %v", err)
	}
}

func TestResolveTitleIDByName_EmptyCatalogIsNotFound(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{}}
	if _, err := resolveTitleIDByName(context.Background(), l, "No Such App"); !errors.Is(err, errTitleNotInCatalog) {
		t.Fatalf("expected errTitleNotInCatalog, got %v", err)
	}
}

// Two titles sharing one exact name must be an ambiguity error, not an
// arbitrary pick — the chosen ID would flip between plans.
func TestResolveTitleIDByName_AmbiguousIsError(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{
		{ID: "A", TitleName: "Duplicated"},
		{ID: "B", TitleName: "Duplicated"},
	}}
	_, err := resolveTitleIDByName(context.Background(), l, "Duplicated")
	if err == nil {
		t.Fatal("expected an error for two titles sharing a name")
	}
	if errors.Is(err, errTitleNotInCatalog) {
		t.Errorf("ambiguity must not be reported as not-found: %v", err)
	}
}

func TestResolveTitleIDByName_TransportErrorPropagates(t *testing.T) {
	want := errors.New("connection refused")
	l := &stubTitleLister{err: want}
	if _, err := resolveTitleIDByName(context.Background(), l, "Jamf Composer"); !errors.Is(err, want) {
		t.Fatalf("expected the transport error to propagate, got %v", err)
	}
}

func TestResolveTitleIDByName_NoLookupForEmptyNameOrNilLister(t *testing.T) {
	l := &stubTitleLister{titles: []pro.AppTitle{{ID: "Composer", TitleName: "Jamf Composer"}}}
	if _, err := resolveTitleIDByName(context.Background(), l, ""); !errors.Is(err, errTitleNotInCatalog) {
		t.Errorf("empty name must be not-found, got %v", err)
	}
	if l.calls != 0 {
		t.Errorf("empty name must not reach the API, got %d calls", l.calls)
	}
	if _, err := resolveTitleIDByName(context.Background(), nil, "Jamf Composer"); err == nil {
		t.Error("nil lister must error")
	}
}

// stubDeploymentLister mirrors stubTitleLister for the deployment resolver.
type stubDeploymentLister struct {
	deployments []pro.AppTitleDeploymentSummary
	err         error
	filter      string
}

func (s *stubDeploymentLister) ListAppInstallerDeploymentsV1(ctx context.Context, sort []string, filter string) ([]pro.AppTitleDeploymentSummary, error) {
	s.filter = filter
	return s.deployments, s.err
}

func TestResolveDeploymentIDByName_ExactMatch(t *testing.T) {
	l := &stubDeploymentLister{deployments: []pro.AppTitleDeploymentSummary{{ID: "9", Name: "tf-acc-composer"}}}
	id, err := resolveDeploymentIDByName(context.Background(), l, "tf-acc-composer")
	if err != nil {
		t.Fatalf("expected the exact name to resolve, got %v", err)
	}
	if id != "9" {
		t.Errorf("id = %q, want 9", id)
	}
	if l.filter != `name=="tf-acc-composer"` {
		t.Errorf("filter = %q", l.filter)
	}
}

// Wire-verified: `name=="TF-PROBE-APPINST-SC"` returns the deployment named
// `tf-probe-appinst-sc`, so the resolver cannot take the server's word for it.
func TestResolveDeploymentIDByName_RejectsCaseMismatch(t *testing.T) {
	l := &stubDeploymentLister{deployments: []pro.AppTitleDeploymentSummary{{ID: "9", Name: "tf-acc-composer"}}}
	if _, err := resolveDeploymentIDByName(context.Background(), l, "TF-ACC-COMPOSER"); !errors.Is(err, errDeploymentNotFound) {
		t.Fatalf("expected errDeploymentNotFound for an off-casing name, got %v", err)
	}
}

func TestResolveDeploymentIDByName_AmbiguousIsError(t *testing.T) {
	l := &stubDeploymentLister{deployments: []pro.AppTitleDeploymentSummary{
		{ID: "9", Name: "dup"},
		{ID: "10", Name: "dup"},
	}}
	_, err := resolveDeploymentIDByName(context.Background(), l, "dup")
	if err == nil {
		t.Fatal("expected an error for two deployments sharing a name")
	}
	if errors.Is(err, errDeploymentNotFound) {
		t.Errorf("ambiguity must not be reported as not-found: %v", err)
	}
}

func TestEscapeRSQLString(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`Jamf Composer`, `Jamf Composer`},
		{`say "hi"`, `say \"hi\"`},
		{`back\slash`, `back\\slash`},
		{`both\ and "`, `both\\ and \"`},
		{`Jamf*`, `Jamf*`}, // an asterisk is left alone: the exact check narrows it
	} {
		if got := escapeRSQLString(tc.in); got != tc.want {
			t.Errorf("escapeRSQLString(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
