// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"fmt"
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
	"github.com/jamf/terraform-provider-jamfplatform/internal/common/egressip"
)

// EgressIPLookupURL is named in the diagnostic as the command to run when the
// lookup itself could not reach the echo service.
const EgressIPLookupURL = egressip.LookupURL

// egressIPLookup reports this host's public source address, indirected so the
// tests below assert on the rendered guidance without network access. Caching
// lives in egressip.Lookup, so the burst an edge block would otherwise cause is
// already handled there.
var egressIPLookup = egressip.Lookup

// edgeBlockPreamble states what happened, in the order an operator needs it:
// the request did not arrive, what answered instead, and that nothing was
// written. The last clause matters most — an apply that failed part-way is the
// first thing they will worry about.
const edgeBlockPreamble = "The request did not reach the Jamf API. A CDN, firewall or IP allowlist answered " +
	"instead; no changes were made.\n\n"

// edgeBlockKnownAddress is used when the lookup succeeded, which is the common
// case: the echo service is not the host being blocked, so a Jamf-side
// allowlist refusing this caller does not stop it answering.
const edgeBlockKnownAddress = "Egress IP address: %s\n\n" + edgeBlockSteps

// edgeBlockUnknownAddress is the fallback. It prints the command rather than
// omitting the address, because an operator who cannot reach the echo service
// from this host can still run it from somewhere sharing the same egress.
const edgeBlockUnknownAddress = "Egress IP address: run `curl -s " + EgressIPLookupURL + "`\n\n" + edgeBlockSteps

// edgeBlockSteps is the action, one step per party who can take it.
//
// Two steps because the page could have come from either end and the summary
// above cannot tell the operator which. Their own proxy or firewall
// intercepting the request is theirs to find; a Jamf-side allowlist refusing
// this address is not something they can inspect, so handing the address over is
// the whole of their part. An earlier draft told them to "check whether it is
// allowed", which is the half they have no way to do.
const edgeBlockSteps = "1. Confirm with your network team that outbound traffic to the Jamf API is not intercepted.\n" +
	"2. Provide the egress IP address, status and request id to Jamf Support."

// gatewayFailureGuidance covers a 5xx page, which needs the opposite remedy to a
// block: the SDK has already retried it, so the answer is to run again rather
// than to go hunting for an allowlist. The split is the one the SDK's godoc asks
// consumers to make.
const gatewayFailureGuidance = "The request did not reach the Jamf API. The gateway answered with an error page; " +
	"no changes were made.\n\n" +
	"The provider has already retried. Re-run the command in a few minutes. Credentials and network " +
	"access are not the cause."

// APIErrorDetail renders err as the detail of a Terraform diagnostic, appending
// what to do about it when the response came from something other than Jamf.
//
// Every resource reporting an API failure should pass its error through here
// rather than calling err.Error() directly. It is a no-op for an error any Jamf
// service produced, which is nearly all of them, so it is applied uniformly
// instead of at the call sites judged likely to see a block: which call hits an
// edge is not a property of the resource, and the two failures that prompted
// this — CloudFront 502 and 504 pages on CI runs 34471595582, 34202213638 and
// 34128074623 — landed on creates, the operations a narrower sweep would have
// skipped.
//
// The classification is IsEdgeBlocked's, and so the SDK's. What the operator
// sees without this is a condensed one-line page summary carrying a status and
// an edge request id — accurate, and no help at all in deciding whether to
// re-run, check an allowlist, or look for a mistake in the configuration.
func APIErrorDetail(err error) string {
	if err == nil {
		return ""
	}
	if !IsEdgeBlocked(err) {
		return err.Error()
	}

	if apiErr := jamfplatform.AsAPIError(err); apiErr != nil && apiErr.StatusCode >= http.StatusInternalServerError {
		return err.Error() + "\n\n" + gatewayFailureGuidance
	}

	address := edgeBlockUnknownAddress
	if ip := egressIPLookup(); ip != "" {
		address = fmt.Sprintf(edgeBlockKnownAddress, ip)
	}
	return err.Error() + "\n\n" + edgeBlockPreamble + address
}
