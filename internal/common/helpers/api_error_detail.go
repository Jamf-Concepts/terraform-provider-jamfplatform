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

// edgeBlockGuidance names the remedy for a standing block: the address an
// allowlist or WAF rule is written against.
//
// The address is printed as a command rather than looked up, which is the one
// place this differs from the provider's authentication diagnostic. That one
// runs once per apply and can afford three seconds of network. This runs
// wherever a resource reports an API failure, and an edge block fails every
// in-flight resource at once, so a lookup here would multiply one outage into a
// burst of third-party calls on a path the operator is already waiting on.
const edgeBlockPreamble = "A CDN, firewall or IP allowlist answered instead of the API, so the request never " +
	"reached the service and nothing changed.\n\n"

// edgeBlockKnownAddress is used when the lookup succeeded, which is the common
// case: the echo service is not the host being blocked, so a Jamf-side
// allowlist refusing this caller does not stop it answering.
//
// It names two owners because the page could have come from either end and the
// operator cannot tell which from the summary above. Their own proxy or firewall
// intercepting the request is theirs to find; a Jamf-side allowlist refusing
// this address is not something they can inspect, so all they can do is hand the
// address over. An earlier draft told them to "check whether it is allowed",
// which is the half they have no way to do.
const edgeBlockKnownAddress = "This host's public IP address is %s. Ask your network team whether outbound traffic " +
	"to the Jamf API is being intercepted, and give the address to Jamf Support to check against the allowlist."

// edgeBlockUnknownAddress is the fallback. It prints a command rather than
// omitting the address, because an operator who cannot reach the echo service
// from this host can still run the command from somewhere that shares its
// egress — and because the same restriction that blocked the lookup is itself a
// clue about what is in front of Jamf.
const edgeBlockUnknownAddress = "Find this host's public IP address with `curl -s " + EgressIPLookupURL +
	"`. Ask your network team whether outbound traffic to the Jamf API is being intercepted, and give the " +
	"address to Jamf Support to check against the allowlist."

// gatewayFailureGuidance covers a 5xx page, which needs the opposite remedy to a
// block: the SDK has already retried it, so the answer is to run again rather
// than to go hunting for an allowlist. The split is the one the SDK's godoc asks
// consumers to make.
const gatewayFailureGuidance = "The gateway answered with an error page, so the request never reached the service " +
	"and nothing changed.\n\n" +
	"The provider retried and got the same page. Run the command again in a few minutes. Your credentials " +
	"and network access are fine."

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
