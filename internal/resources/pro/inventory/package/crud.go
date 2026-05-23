// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

// SDK endpoints used:
//   pro.CreatePackageV1
//   pro.GetPackageV1
//   pro.UpdatePackageV1
//   pro.DeletePackageV1
//   pro.UploadPackageV1
//   pro.UploadPackageManifestV1
//   pro.DeletePackageManifestV1
//   pro.RefreshCloudDistributionPointInventoryV1
//   pro.ListPackagesV1 (data source / list resource)
//   pro.ResolvePackageV1ByName (data source)
//
// Status: current. Last reviewed 2026-05-23.

package pkg

import (
	"context"
	"crypto/sha3"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/pro"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/files"
	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// Create creates a Jamf Pro package. The high-level flow:
//
//  1. POST metadata via CreatePackageV1 to obtain an ID.
//  2. If `package_file_source` is set (JCDS mode): open the source once,
//     hash + checksum-validate, rewind, stream to UploadPackageV1, then
//     poll until JCDS converges. The server populates every hash field
//     and `cloudTransferStatus`.
//  3. If `manifest_file_source` is set: POST it to the /manifest sub-resource.
//  4. Final GET to refresh state with server-populated fields.
//
// FSDP modes skip step 2 (and step 3 when no manifest source is supplied).
func (r *PackageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan PackageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultCreateTimeout, plan.Timeouts.Create)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	createCtx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	created, err := r.client.CreatePackageV1(createCtx, buildPackageInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating Jamf Pro package", err.Error())
		return
	}
	if created == nil || created.ID == "" {
		resp.Diagnostics.AddError(
			"Create response missing package ID",
			"Jamf Pro returned 201 Created with no ID; cannot persist state.",
		)
		return
	}
	plan.ID = types.StringValue(created.ID)

	if isConfiguredString(plan.PackageFileSource) {
		if streamingURLEnabled(plan) {
			if diags := streamURLUploadAndVerify(createCtx, r.client, &plan, ""); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		} else {
			if diags := openHashUploadVerify(createCtx, r.client, &plan, ""); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	}

	if isConfiguredString(plan.ManifestFileSource) {
		if diags := uploadPackageManifest(createCtx, r.client, &plan); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	got, err := r.client.GetPackageV1(createCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading created Jamf Pro package", err.Error())
		return
	}
	resp.Diagnostics.Append(assignPackageResourceModel(&plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, packageIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created Jamf Pro package", map[string]any{"id": plan.ID.ValueString()})
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read refreshes Terraform state with the latest package representation.
func (r *PackageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state PackageResourceModel
	isImport := req.State.Raw.IsNull()

	if isImport {
		if req.Identity == nil {
			resp.Diagnostics.AddError(
				"Missing resource identity",
				"Terraform requested a refresh for this package without existing state or identity data, so the provider cannot determine which package to read.",
			)
			return
		}
		var identity packageIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if identity.ID.IsNull() || identity.ID.IsUnknown() || identity.ID.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Missing package ID",
				"The resource identity did not include an 'id' attribute, so the provider cannot refresh the package.",
			)
			return
		}
		state.ID = identity.ID
		state.Timeouts = helpers.NewResourceTimeoutsNullValue(packageTimeoutAttributeTypes)
	} else {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	readTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultReadTimeout, state.Timeouts.Read)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	readCtx, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot read Jamf Pro package without ID.")
		return
	}

	got, err := r.client.GetPackageV1(readCtx, state.ID.ValueString())
	if err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro package not found, removing from state", map[string]any{"id": state.ID.ValueString()})
			resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, packageIdentityModel{ID: state.ID})...)
			if resp.Diagnostics.HasError() {
				return
			}
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Jamf Pro package", err.Error())
		return
	}

	resp.Diagnostics.Append(assignPackageResourceModel(&state, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, packageIdentityModel{ID: state.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies a metadata PUT and (when re-upload conditions hold) streams
// the new binary and re-runs the verification poll. The PUT is full-replace
// (§13.8.A.7), so buildPackageInput emits the full intended state. Manifest
// re-upload is direct-equality gated; deletion fires when the source
// transitions from set → null.
//
// JCDS source-handling: the source is opened exactly once. We hash from the
// open handle, validate any user-supplied checksum, decide whether the local
// bytes differ from state's recorded sha3, and (if re-upload is needed)
// rewind the SAME handle and stream it to UploadPackageV1. URL sources are
// downloaded exactly once per Update.
func (r *PackageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state PackageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, plan.Timeouts.IsNull(), plan.Timeouts.IsUnknown(), defaultUpdateTimeout, plan.Timeouts.Update)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	updateCtx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	if plan.ID.IsNull() || plan.ID.IsUnknown() || plan.ID.ValueString() == "" {
		plan.ID = state.ID
	}

	// Streaming-URL mode short-circuits the disk-staging path entirely.
	// The metadata PUT runs first, then the streaming upload, then manifest
	// reconciliation, then the final GET. No local hash compare gate — the
	// streaming path always uploads (it has no local hash to compare against
	// state until after the stream completes).
	if streamingURLEnabled(plan) {
		// Upload + verify FIRST so JCDS has recomputed hashes when the
		// metadata PUT lands (see disk-staging path for full rationale).
		previous := strings.ToLower(state.Sha3512.ValueString())
		if diags := streamURLUploadAndVerify(updateCtx, r.client, &plan, previous); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		// Echo server state into PUT body so the full-replace does not
		// silently drop any field. mergePlanIntoServerState layers user
		// metadata on top of the canonical server record.
		refreshed, getErr := r.client.GetPackageV1(updateCtx, plan.ID.ValueString())
		if getErr != nil {
			resp.Diagnostics.AddError("Error reading post-upload Jamf Pro package", getErr.Error())
			return
		}
		tflog.Info(ctx, "streaming post-poll GET", map[string]any{
			"id":         plan.ID.ValueString(),
			"got_hash":   helpers.DerefString(refreshed.HashValue),
			"got_status": helpers.DerefString(refreshed.CloudTransferStatus),
			"got_size":   helpers.DerefString(refreshed.Size),
		})
		streamInput := mergePlanIntoServerState(plan, refreshed)
		tflog.Info(ctx, "streaming metadata PUT body", map[string]any{
			"id":         plan.ID.ValueString(),
			"input_info": helpers.DerefString(streamInput.Info),
			"input_hash": helpers.DerefString(streamInput.HashValue),
			"input_size": helpers.DerefString(streamInput.Size),
		})
		streamPutResp, err := r.client.UpdatePackageV1(updateCtx, plan.ID.ValueString(), streamInput)
		if err != nil {
			resp.Diagnostics.AddError("Error updating Jamf Pro package", err.Error())
			return
		}
		tflog.Info(ctx, "streaming metadata PUT response", map[string]any{
			"id":        plan.ID.ValueString(),
			"resp_hash": helpers.DerefString(streamPutResp.HashValue),
			"resp_size": helpers.DerefString(streamPutResp.Size),
		})
		if diags := reconcileManifestAndFinalise(updateCtx, ctx, r.client, &plan, &state, resp); diags.HasError() {
			resp.Diagnostics.Append(diags...)
		}
		return
	}

	// Disk-staging path (default): open the source once, hash, decide re-upload.
	var (
		uploadFile   *os.File
		uploadName   string
		cleanup      = func() {}
		willReupload bool
		localSha3    string
		localSize    int64
		previousHash string
	)
	if isConfiguredString(plan.PackageFileSource) {
		f, filename, c, openErr := files.OpenUploadSource(updateCtx, plan.PackageFileSource.ValueString(), files.DefaultMaxBytes)
		if openErr != nil {
			resp.Diagnostics.AddError("Error opening package binary", openErr.Error())
			return
		}
		cleanup = c

		sha, sz, hashErr := HashStreamSHA3(f)
		if hashErr != nil {
			cleanup()
			resp.Diagnostics.AddError("Error hashing package binary", hashErr.Error())
			return
		}
		localSha3 = sha
		localSize = sz

		if isConfiguredString(plan.PackageFileSourceChecksum) {
			declared := strings.ToLower(plan.PackageFileSourceChecksum.ValueString())
			if declared != strings.ToLower(localSha3) {
				cleanup()
				resp.Diagnostics.AddError(
					"package_file_source_checksum did not match local SHA-3-512",
					fmt.Sprintf("%v: expected %s, computed %s", errLocalChecksumMismatch, declared, localSha3),
				)
				return
			}
		}

		stateSha := strings.ToLower(state.Sha3512.ValueString())
		willReupload = stateSha == "" || stateSha != strings.ToLower(localSha3)
		previousHash = stateSha

		tflog.Info(ctx, "package update reupload decision", map[string]any{
			"state_sha3":    stateSha,
			"local_sha3":    localSha3,
			"will_reupload": willReupload,
			"source":        plan.PackageFileSource.ValueString(),
			"file_name":     plan.FileName.ValueString(),
			"id":            plan.ID.ValueString(),
		})

		if willReupload {
			uploadFile = f
			uploadName = filename
		} else {
			cleanup()
			cleanup = func() {}
		}
	}
	defer cleanup()

	// Binary re-upload + verification FIRST (before metadata PUT). Spike
	// §13.3 / §13.5: the server does not validate user-supplied hashValue
	// PUTs, so injecting localSha3 into a pre-upload PUT body silently
	// short-circuits the verification poll — the poll sees the user-PUT
	// hash on the very first tick and declares convergence while JCDS is
	// still serving the old binary. Upload first, let JCDS recompute,
	// then PUT metadata so the post-PUT GET reflects the canonical
	// server state.
	if willReupload {
		if _, seekErr := uploadFile.Seek(0, io.SeekStart); seekErr != nil {
			resp.Diagnostics.AddError("Error preparing package upload", seekErr.Error())
			return
		}
		fileName := plan.FileName.ValueString()
		if fileName == "" {
			fileName = uploadName
		}
		if diags := uploadAndPoll(updateCtx, r.client, plan.ID.ValueString(), fileName, uploadFile, localSha3, localSize, previousHash); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
	}

	// Manifest reconciliation runs BEFORE the metadata PUT. Discovered via
	// ManifestLifecycle acc trace: DeletePackageManifestV1 has a server-side
	// side effect that clears the package's `size` field. Running the
	// metadata PUT afterwards lets us echo state's size value back into the
	// record so the post-PUT GET reflects the canonical full state.
	planHasManifestSrc := isConfiguredString(plan.ManifestFileSource)
	stateHadManifest := !state.Manifest.IsNull() && state.Manifest.ValueString() != ""

	switch {
	case planHasManifestSrc:
		equal, eqErr := ManifestBodiesEqual(updateCtx, state.Manifest.ValueString(), plan.ManifestFileSource.ValueString())
		if eqErr != nil {
			resp.Diagnostics.AddError("Error comparing package manifest source", eqErr.Error())
			return
		}
		if !equal {
			if diags := uploadPackageManifest(updateCtx, r.client, &plan); diags.HasError() {
				resp.Diagnostics.Append(diags...)
				return
			}
		}
	case stateHadManifest && !planHasManifestSrc:
		if err := r.client.DeletePackageManifestV1(updateCtx, plan.ID.ValueString()); err != nil {
			if !helpers.IsNotFoundError(err) {
				resp.Diagnostics.AddError("Error deleting Jamf Pro package manifest", err.Error())
				return
			}
		}
		// DeletePackageManifestV1 clears the package's server-derived
		// `size` field (observed via ManifestLifecycle acc trace). Nudge
		// the cloud distribution point so JCDS re-derives size from the
		// binary that's still attached to this package id. Errors are
		// non-fatal — the worst case is a slightly stale state.
		if state.FileName.ValueString() != "" {
			_ = r.client.RefreshCloudDistributionPointInventoryV1(updateCtx, state.FileName.ValueString())
		}
	}

	// Metadata PUT (full-replace) — runs after upload+poll AND manifest
	// reconcile. Spike §13.8 A.7 confirmed PUT is full-replace; we GET
	// the canonical record first so every server-derived field that the
	// resource does not own survives. Matches the spike's S5 probe pattern.
	postReconcile, err := r.client.GetPackageV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading post-reconcile Jamf Pro package", err.Error())
		return
	}
	input := mergePlanIntoServerState(plan, postReconcile)
	if willReupload {
		sha3TypeTag := hashTypeSHA3512
		input.HashType = &sha3TypeTag
		input.HashValue = &localSha3
		input.Sha3512 = &localSha3
	}
	tflog.Info(ctx, "package metadata PUT body", map[string]any{
		"id":          plan.ID.ValueString(),
		"input_info":  helpers.DerefString(input.Info),
		"input_notes": helpers.DerefString(input.Notes),
		"input_hash":  helpers.DerefString(input.HashValue),
		"input_size":  helpers.DerefString(input.Size),
	})
	putResp, err := r.client.UpdatePackageV1(updateCtx, plan.ID.ValueString(), input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Jamf Pro package", err.Error())
		return
	}
	tflog.Info(ctx, "package metadata PUT response", map[string]any{
		"id":         plan.ID.ValueString(),
		"resp_info":  helpers.DerefString(putResp.Info),
		"resp_notes": helpers.DerefString(putResp.Notes),
		"resp_hash":  helpers.DerefString(putResp.HashValue),
		"resp_size":  helpers.DerefString(putResp.Size),
	})

	got, err := r.client.GetPackageV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading updated Jamf Pro package", err.Error())
		return
	}
	tflog.Info(ctx, "package update final GET", map[string]any{
		"id":        plan.ID.ValueString(),
		"got_info":  helpers.DerefString(got.Info),
		"got_notes": helpers.DerefString(got.Notes),
		"got_hash":  helpers.DerefString(got.HashValue),
		"got_size":  helpers.DerefString(got.Size),
	})
	resp.Diagnostics.Append(assignPackageResourceModel(&plan, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(helpers.SetIdentity(ctx, resp.Identity, packageIdentityModel{ID: plan.ID})...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes a Jamf Pro package.
func (r *PackageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state PackageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteTimeout, timeoutDiags := helpers.ResolveTimeout(ctx, state.Timeouts.IsNull(), state.Timeouts.IsUnknown(), defaultDeleteTimeout, state.Timeouts.Delete)
	resp.Diagnostics.Append(timeoutDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	deleteCtx, cancel := context.WithTimeout(ctx, deleteTimeout)
	defer cancel()

	if state.ID.IsNull() || state.ID.ValueString() == "" {
		resp.Diagnostics.AddError("Missing ID", "Cannot delete Jamf Pro package without ID.")
		return
	}

	if err := r.client.DeletePackageV1(deleteCtx, state.ID.ValueString()); err != nil {
		if helpers.IsNotFoundError(err) {
			tflog.Info(ctx, "Jamf Pro package already removed", map[string]any{"id": state.ID.ValueString()})
			return
		}
		resp.Diagnostics.AddError("Error deleting Jamf Pro package", fmt.Sprintf("API error: %v", err))
	}
}

// isConfiguredString reports whether a Terraform string attribute has a
// non-null, non-unknown, non-empty user-supplied value.
func isConfiguredString(v types.String) bool {
	return helpers.IsConfiguredValue(v) && v.ValueString() != ""
}

// openHashUploadVerify drives the JCDS upload + verification poll for
// Create. Opens the source exactly once, hashes from the open handle,
// validates the optional user-supplied checksum, rewinds, streams to
// UploadPackageV1, and polls until convergence. previousHash is empty
// on Create.
func openHashUploadVerify(ctx context.Context, client *pro.Client, plan *PackageResourceModel, previousHash string) diag.Diagnostics {
	file, filename, cleanup, err := files.OpenUploadSource(ctx, plan.PackageFileSource.ValueString(), files.DefaultMaxBytes)
	if err != nil {
		return errorDiag("Error opening package binary", err.Error())
	}
	defer cleanup()

	sha, sz, hashErr := HashStreamSHA3(file)
	if hashErr != nil {
		return errorDiag("Error hashing package binary", hashErr.Error())
	}

	if isConfiguredString(plan.PackageFileSourceChecksum) {
		declared := strings.ToLower(plan.PackageFileSourceChecksum.ValueString())
		if declared != strings.ToLower(sha) {
			return errorDiag(
				"package_file_source_checksum did not match local SHA-3-512",
				fmt.Sprintf("%v: expected %s, computed %s", errLocalChecksumMismatch, declared, sha),
			)
		}
	}

	if _, seekErr := file.Seek(0, io.SeekStart); seekErr != nil {
		return errorDiag("Error preparing package upload", seekErr.Error())
	}

	fileName := plan.FileName.ValueString()
	if fileName == "" {
		fileName = filename
	}
	return uploadAndPoll(ctx, client, plan.ID.ValueString(), fileName, file, sha, sz, previousHash)
}

// uploadAndPoll streams the supplied (already-rewound) file to
// UploadPackageV1, then drives the verification poll. Splitting this from
// the open/hash phase lets Update reuse the open handle from the hash
// decision.
func uploadAndPoll(ctx context.Context, client *pro.Client, id, fileName string, file io.Reader, localSha3 string, localSize int64, previousHash string) diag.Diagnostics {
	tflog.Info(ctx, "package upload starting", map[string]any{
		"id":            id,
		"file_name":     fileName,
		"local_sha3":    localSha3,
		"local_size":    localSize,
		"previous_hash": previousHash,
	})
	href, err := client.UploadPackageV1(ctx, id, fileName, file)
	if err != nil {
		tflog.Error(ctx, "package upload failed", map[string]any{"err": err.Error()})
		return errorDiag("Error uploading Jamf Pro package binary", err.Error())
	}
	hrefStr := ""
	if href != nil {
		hrefStr = href.Href
	}
	tflog.Info(ctx, "package upload returned", map[string]any{"id": id, "href": hrefStr})

	convergedPkg, pollErr := PollPackageVerification(ctx, client, id, fileName, localSha3, localSize, previousHash)
	if pollErr != nil {
		tflog.Error(ctx, "package poll failed", map[string]any{"err": pollErr.Error()})
		if errors.Is(pollErr, errCorruption) {
			return errorDiag(
				"Package binary verification failed (corruption)",
				pollErr.Error(),
			)
		}
		if errors.Is(pollErr, errVerificationTimeout) {
			return errorDiag(
				"Package binary verification timed out",
				"JCDS did not converge on the expected hash before the timeout fired. Increase `timeouts.create` (or `timeouts.update`) to allow more time, or check the Jamf Pro admin console for CDP status.",
			)
		}
		return errorDiag("Error verifying Jamf Pro package upload", pollErr.Error())
	}
	convergedHash := ""
	convergedStatus := ""
	if convergedPkg != nil {
		if convergedPkg.HashValue != nil {
			convergedHash = *convergedPkg.HashValue
		}
		if convergedPkg.CloudTransferStatus != nil {
			convergedStatus = *convergedPkg.CloudTransferStatus
		}
	}
	tflog.Info(ctx, "package poll converged", map[string]any{
		"id":                  id,
		"converged_hash":      convergedHash,
		"converged_status":    convergedStatus,
		"expected_local_sha3": localSha3,
	})
	return nil
}

// uploadPackageManifest streams the manifest source to the /manifest
// sub-resource. Used by both Create and Update — the upload response
// returns the updated Package but state is refreshed later via the final
// GET so we discard it here.
func uploadPackageManifest(ctx context.Context, client *pro.Client, plan *PackageResourceModel) diag.Diagnostics {
	src := plan.ManifestFileSource.ValueString()

	file, filename, cleanup, err := files.OpenUploadSource(ctx, src, files.DefaultMaxBytes)
	if err != nil {
		return errorDiag("Error opening manifest source", err.Error())
	}
	defer cleanup()

	if _, err := client.UploadPackageManifestV1(ctx, plan.ID.ValueString(), filename, file); err != nil {
		return errorDiag("Error uploading Jamf Pro package manifest", err.Error())
	}
	return nil
}

// errorDiag is a small constructor for the single-error diag.Diagnostics
// value the upload helpers return.
func errorDiag(summary, detail string) diag.Diagnostics {
	return diag.Diagnostics{diag.NewErrorDiagnostic(summary, detail)}
}

// streamingURLEnabled reports whether the plan opted into the
// stream-the-URL-body-directly path AND the package source is in fact a
// URL. For local-path sources the flag is silently ignored.
func streamingURLEnabled(plan PackageResourceModel) bool {
	if !plan.StreamURLDirectly.ValueBool() {
		return false
	}
	return isConfiguredString(plan.PackageFileSource) && files.URLSource(plan.PackageFileSource.ValueString())
}

// streamURLUploadAndVerify wires up io.Pipe + io.MultiWriter so the URL
// body flows simultaneously into the SHA-3 hasher and the multipart upload
// pipe. The hash is known only AFTER the upload completes; the verification
// poll then runs against the just-computed digest.
//
// Tradeoffs (versus the disk-staging path):
//   - No 429 retry — the upload reader is not seekable.
//   - No pre-upload checksum validation — bytes leave before the hash is known.
//   - No Content-Length precompute — SDK transport falls back to chunked TE.
//   - Mid-stream origin failure aborts the upload; a retry forces a fresh GET.
//
// Trade is intentional: see schema description for the user-facing
// rationale. Caller has already enforced `package_file_source` is a URL
// (streamingURLEnabled guards entry).
func streamURLUploadAndVerify(ctx context.Context, client *pro.Client, plan *PackageResourceModel, previousHash string) diag.Diagnostics {
	body, urlFilename, err := files.OpenURLStream(ctx, plan.PackageFileSource.ValueString(), files.DefaultMaxBytes)
	if err != nil {
		return errorDiag("Error opening package URL stream", err.Error())
	}
	defer func() { _ = body.Close() }()

	pr, pw := io.Pipe()
	hasher := sha3.New512()
	var bytesStreamed int64

	// Drive the URL → (hasher, pipe-writer) tee on a goroutine so the SDK
	// can read from pr concurrently. Errors propagate through
	// pw.CloseWithError so the SDK upload terminates with the right cause.
	go func() {
		n, copyErr := io.Copy(io.MultiWriter(pw, hasher), body)
		bytesStreamed = n
		if copyErr != nil {
			_ = pw.CloseWithError(fmt.Errorf("streaming URL body: %w", copyErr))
			return
		}
		_ = pw.Close()
	}()

	fileName := plan.FileName.ValueString()
	if fileName == "" {
		fileName = urlFilename
	}
	if _, err := client.UploadPackageV1(ctx, plan.ID.ValueString(), fileName, pr); err != nil {
		// Drain any remaining body so the goroutine exits cleanly.
		_ = pr.CloseWithError(err)
		return errorDiag("Error streaming Jamf Pro package upload", err.Error())
	}

	localSha3 := hex.EncodeToString(hasher.Sum(nil))
	localSize := bytesStreamed

	if _, err := PollPackageVerification(ctx, client, plan.ID.ValueString(), fileName, localSha3, localSize, previousHash); err != nil {
		if errors.Is(err, errCorruption) {
			return errorDiag("Package binary verification failed (corruption)", err.Error())
		}
		if errors.Is(err, errVerificationTimeout) {
			return errorDiag(
				"Package binary verification timed out",
				"JCDS did not converge on the expected hash before the timeout fired. Increase `timeouts.create` (or `timeouts.update`) to allow more time, or check the Jamf Pro admin console for CDP status.",
			)
		}
		return errorDiag("Error verifying Jamf Pro package upload", err.Error())
	}
	return nil
}

// reconcileManifestAndFinalise runs the manifest reconciliation and the
// final state refresh for the streaming-URL Update path. Returns
// diagnostics; on success it has already called resp.State.Set.
func reconcileManifestAndFinalise(updateCtx, identityCtx context.Context, client *pro.Client, plan, state *PackageResourceModel, resp *resource.UpdateResponse) diag.Diagnostics {
	planHasManifestSrc := isConfiguredString(plan.ManifestFileSource)
	stateHadManifest := !state.Manifest.IsNull() && state.Manifest.ValueString() != ""

	switch {
	case planHasManifestSrc:
		equal, eqErr := ManifestBodiesEqual(updateCtx, state.Manifest.ValueString(), plan.ManifestFileSource.ValueString())
		if eqErr != nil {
			return errorDiag("Error comparing package manifest source", eqErr.Error())
		}
		if !equal {
			if diags := uploadPackageManifest(updateCtx, client, plan); diags.HasError() {
				return diags
			}
		}
	case stateHadManifest && !planHasManifestSrc:
		if err := client.DeletePackageManifestV1(updateCtx, plan.ID.ValueString()); err != nil {
			if !helpers.IsNotFoundError(err) {
				return errorDiag("Error deleting Jamf Pro package manifest", err.Error())
			}
		}
	}

	got, err := client.GetPackageV1(updateCtx, plan.ID.ValueString())
	if err != nil {
		return errorDiag("Error reading updated Jamf Pro package", err.Error())
	}
	if diags := assignPackageResourceModel(plan, got); diags.HasError() {
		return diags
	}
	if diags := helpers.SetIdentity(identityCtx, resp.Identity, packageIdentityModel{ID: plan.ID}); diags.HasError() {
		return diags
	}
	return resp.State.Set(identityCtx, plan)
}
