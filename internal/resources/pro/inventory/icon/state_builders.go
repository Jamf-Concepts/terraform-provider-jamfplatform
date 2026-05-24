// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package icon

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// sourceHashPrefix is the algorithm tag prepended to every value stored in
// the source_hash attribute. The single uniform format lets the provider
// distinguish provider-computed hashes from arbitrary user-supplied
// strings (legacy or hand-edited state).
const sourceHashPrefix = "sha256:"

// assignIconResourceModel copies the SDK IconResponse fields into state.
func assignIconResourceModel(state *IconResourceModel, resp *pro.IconResponse) {
	state.ID = types.StringValue(strconv.Itoa(resp.ID))
	state.URL = types.StringValue(resp.URL)
}

// computeSourceHashString returns the canonical source_hash string for the
// supplied bytes: "sha256:" + lowercase hex SHA-256.
func computeSourceHashString(b []byte) string {
	sum := sha256.Sum256(b)
	return sourceHashPrefix + hex.EncodeToString(sum[:])
}
