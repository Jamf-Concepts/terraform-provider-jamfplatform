// Copyright Jamf Software LLC 2026
// SPDX-License-Identifier: MPL-2.0

package mobile_device_enrollment_profile

import (
	"github.com/Jamf-Concepts/jamfplatform-go-sdk/jamfplatform/proclassic"

	"github.com/Jamf-Concepts/terraform-provider-jamfplatform/internal/common/helpers"
)

// assignEnrollmentProfileResourceModel refreshes a resource model from a GET.
// Optional nested blocks (location / purchasing) are refreshed only when already
// authored (the model pointer is non-nil) so the always-returned server defaults
// don't fabricate blocks the user never declared. On import the blocks start nil
// and are not populated — ImportStateVerifyIgnore covers them.
func assignEnrollmentProfileResourceModel(state *EnrollmentProfileResourceModel, api *proclassic.MobileDeviceEnrollmentProfile) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Description = helpers.StringPointerValueOrNull(g.Description)
	state.Invitation = bigIntStringOrNull(g.Invitation)
	state.UUID = stringOrNull(g.UUID)
	if g.Site != nil {
		state.SiteID = helpers.StringValueFromIntPtr(g.Site.ID)
		state.SiteName = stringOrNull(g.Site.Name)
	}

	if state.Location != nil {
		state.Location = flattenLocationModel(api.Location)
	}
	if state.Purchasing != nil {
		state.Purchasing = flattenPurchasingModel(api.Purchasing)
	}
	state.Attachments = flattenAttachments(api.Attachments)
}

// assignEnrollmentProfileDataSourceModel populates a DS model from a GET. The DS
// always surfaces location / purchasing / attachments (read-only lookup).
func assignEnrollmentProfileDataSourceModel(state *EnrollmentProfileDataSourceModel, api *proclassic.MobileDeviceEnrollmentProfile) {
	if api == nil || api.General == nil {
		return
	}
	g := api.General
	if g.ID != nil {
		state.ID = helpers.StringValueFromIntPtr(g.ID)
	}
	state.Name = helpers.StringPointerValueOrNull(g.Name)
	state.Description = helpers.StringPointerValueOrNull(g.Description)
	state.Invitation = bigIntStringOrNull(g.Invitation)
	state.UUID = stringOrNull(g.UUID)
	if g.Site != nil {
		state.SiteID = helpers.StringValueFromIntPtr(g.Site.ID)
		state.SiteName = stringOrNull(g.Site.Name)
	}
	state.Location = flattenLocationModel(api.Location)
	state.Purchasing = flattenPurchasingModel(api.Purchasing)
	state.Attachments = flattenAttachments(api.Attachments)
}

func flattenLocationModel(api *proclassic.Location) *LocationModel {
	if api == nil {
		return nil
	}
	return &LocationModel{
		Username:     stringOrNull(api.Username),
		RealName:     stringOrNull(firstNonNil(api.RealName, api.Realname)),
		EmailAddress: stringOrNull(api.EmailAddress),
		PhoneNumber:  stringOrNull(firstNonNil(api.PhoneNumber, api.Phone)),
		Department:   stringOrNull(api.Department),
		Building:     stringOrNull(api.Building),
		Room:         stringOrNull(api.Room),
		Position:     stringOrNull(api.Position),
	}
}

func flattenPurchasingModel(api *proclassic.Purchasing) *PurchasingModel {
	if api == nil {
		return nil
	}
	return &PurchasingModel{
		IsPurchased:       helpers.BoolPointerValueOrNull(api.IsPurchased),
		IsLeased:          helpers.BoolPointerValueOrNull(api.IsLeased),
		PONumber:          stringOrNull(api.PoNumber),
		PODate:            stringOrNull(api.PoDate),
		PODateEpoch:       intStringOrNull(api.PoDateEpoch),
		PODateUTC:         stringOrNull(api.PoDateUtc),
		Vendor:            stringOrNull(api.Vendor),
		WarrantyExpires:   stringOrNull(api.WarrantyExpires),
		WarrantyEpoch:     intStringOrNull(api.WarrantyExpiresEpoch),
		WarrantyUTC:       stringOrNull(api.WarrantyExpiresUtc),
		AppleCareID:       stringOrNull(api.ApplecareID),
		LeaseExpires:      stringOrNull(api.LeaseExpires),
		LeaseEpoch:        intStringOrNull(api.LeaseExpiresEpoch),
		LeaseUTC:          stringOrNull(api.LeaseExpiresUtc),
		PurchasePrice:     stringOrNull(api.PurchasePrice),
		LifeExpectancy:    int64ValueOrNull(api.LifeExpectancy),
		PurchasingAccount: stringOrNull(api.PurchasingAccount),
		PurchasingContact: stringOrNull(api.PurchasingContact),
	}
}

func firstNonNil(a, b *string) *string {
	if a != nil && *a != "" {
		return a
	}
	return b
}
