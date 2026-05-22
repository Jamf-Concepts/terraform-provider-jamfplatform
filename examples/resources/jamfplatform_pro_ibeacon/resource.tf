resource "jamfplatform_pro_ibeacon" "reception" {
  name  = "Reception"
  uuid  = "759b0599-64e0-416a-8d31-d8e93482a4d7"
  major = 1
  minor = 2
}

resource "jamfplatform_pro_ibeacon" "match_any" {
  name                    = "Lobby — any major/minor"
  uuid                    = "759b0599-64e0-416a-8d31-d8e93482a4d7"
  include_any_major_value = true
  include_any_minor_value = true
}

resource "jamfplatform_pro_ibeacon" "specific_major_any_minor" {
  name                    = "Floor 3 — any beacon"
  uuid                    = "759b0599-64e0-416a-8d31-d8e93482a4d7"
  major                   = 42
  include_any_minor_value = true
}
