// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package app_installer

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
)

// errTitleNotInCatalog reports that the App Catalog holds no title whose display
// name equals the requested one exactly. It is the resolver's own verdict rather
// than a server status, because Jamf Pro answers a name that matches nothing with
// an empty list and a 200.
var errTitleNotInCatalog = errors.New("no App Catalog title has that exact name")

// titleLister is the subset of *pro.Client the App Catalog name resolver uses.
// Declaring it as an interface keeps the resolver unit-testable without a live
// client.
type titleLister interface {
	ListAppInstallerTitlesV1(ctx context.Context, sort []string, filter string) ([]pro.AppTitle, error)
}

// resolveTitleIDByName resolves an App Catalog title display name to its catalog
// ID, matching the name EXACTLY.
//
// Jamf Pro's own `titleName` filter cannot be trusted to decide the match on its
// own: it is a case-insensitive glob, so `titleName=="jamf composer"` and
// `titleName=="Jamf*"` both match "Jamf Composer". Accepting either would store
// the user's spelling, then reverse-resolve it to the canonical name on the next
// Read and produce a perpetual diff — the trap
// STYLE_GUIDE §"Referencing a server-managed catalog by name" warns about. So the
// filter is used only to narrow the request, and the match is decided here on
// byte equality.
//
// A name that matches nothing, or matches only case-insensitively, returns
// errTitleNotInCatalog; two titles sharing one exact name return an ambiguity
// error rather than an arbitrary pick.
func resolveTitleIDByName(ctx context.Context, lister titleLister, name string) (string, error) {
	if lister == nil {
		return "", errors.New("no App Catalog client configured")
	}
	if name == "" {
		return "", errTitleNotInCatalog
	}

	candidates, err := lister.ListAppInstallerTitlesV1(ctx, nil, titleNameFilter(name))
	if err != nil {
		return "", err
	}

	var matches []pro.AppTitle
	for _, t := range candidates {
		if t.TitleName == name {
			matches = append(matches, t)
		}
	}

	switch len(matches) {
	case 0:
		return "", errTitleNotInCatalog
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, 0, len(matches))
		for _, m := range matches {
			ids = append(ids, m.ID)
		}
		return "", fmt.Errorf("the App Catalog holds %d titles named %q (IDs %s); reference one by ID instead", len(matches), name, strings.Join(ids, ", "))
	}
}

// titleNameFilter renders an RSQL equality filter over a title display name.
func titleNameFilter(name string) string {
	return fmt.Sprintf(`titleName=="%s"`, escapeRSQLString(name))
}

// errDeploymentNotFound reports that no deployment carries the requested name
// exactly. Like the title resolver, this is the resolver's own verdict: Jamf Pro
// answers a name that matches nothing with an empty list and a 200.
var errDeploymentNotFound = errors.New("no App Installer deployment has that exact name")

// deploymentLister is the subset of *pro.Client the deployment name resolver
// uses.
type deploymentLister interface {
	ListAppInstallerDeploymentsV1(ctx context.Context, sort []string, filter string) ([]pro.AppTitleDeploymentSummary, error)
}

// resolveDeploymentIDByName resolves a deployment name to its ID, matching the
// name EXACTLY for the same reason as resolveTitleIDByName: Jamf Pro's `name`
// filter is a case-insensitive glob, so the filter narrows the request and the
// match is decided here on byte equality. Two deployments may legitimately share
// a name, which is an ambiguity error rather than an arbitrary pick.
func resolveDeploymentIDByName(ctx context.Context, lister deploymentLister, name string) (string, error) {
	if lister == nil {
		return "", errors.New("no App Installer client configured")
	}
	if name == "" {
		return "", errDeploymentNotFound
	}

	candidates, err := lister.ListAppInstallerDeploymentsV1(ctx, nil, deploymentNameFilter(name))
	if err != nil {
		return "", err
	}

	var ids []string
	for _, d := range candidates {
		if d.Name == name {
			ids = append(ids, d.ID)
		}
	}

	switch len(ids) {
	case 0:
		return "", errDeploymentNotFound
	case 1:
		return ids[0], nil
	default:
		return "", fmt.Errorf("%d App Installer deployments are named %q (IDs %s); look one up by id instead", len(ids), name, strings.Join(ids, ", "))
	}
}

// deploymentNameFilter renders an RSQL equality filter over a deployment name.
func deploymentNameFilter(name string) string {
	return fmt.Sprintf(`name=="%s"`, escapeRSQLString(name))
}

// escapeRSQLString escapes a value for an RSQL double-quoted string, so a
// backslash or double quote in a name cannot break the query. An asterisk is
// left alone: Jamf Pro reads it as a glob, and both callers decide the match on
// exact equality anyway, so a widened candidate set is harmless.
func escapeRSQLString(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	return strings.ReplaceAll(v, `"`, `\"`)
}
