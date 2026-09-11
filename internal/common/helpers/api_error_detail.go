// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package helpers

import (
	"net/http"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform"
)

// EgressIPLookupURL echoes the caller's public source address as a bare line of
// text, which is the address a Jamf-side allowlist or WAF rule is written
// against. Shared with the provider's authentication diagnostic so the two
// cannot name different services.
const EgressIPLookupURL = "https://checkip.amazonaws.com"

// edgeBlockGuidance names the remedy for a standing block: the address an
// allowlist or WAF rule is written against.
//
// The address is printed as a command rather than looked up, which is the one
// place this differs from the provider's authentication diagnostic. That one
// runs once per apply and can afford three seconds of network. This runs
// wherever a resource reports an API failure, and an edge block fails every
// in-flight resource at once, so a lookup here would multiply one outage into a
// burst of third-party calls on a path the operator is already waiting on.
const edgeBlockGuidance = "A CDN, firewall or IP allowlist answered instead of the API, so the request never " +
	"reached the service and nothing changed.\n\n" +
	"Check whether this host's IP address is allowed. Find the address with `curl -s " + EgressIPLookupURL +
	"`, then give it to Jamf Support with the status and request id above."

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

	guidance := edgeBlockGuidance
	if apiErr := jamfplatform.AsAPIError(err); apiErr != nil && apiErr.StatusCode >= http.StatusInternalServerError {
		guidance = gatewayFailureGuidance
	}
	return err.Error() + "\n\n" + guidance
}
