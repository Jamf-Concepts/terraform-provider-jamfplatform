resource "jamfplatform_pro_network_segment" "hq" {
  name             = "HQ Office"
  starting_address = "10.0.0.0"
  ending_address   = "10.0.0.255"
}

resource "jamfplatform_pro_network_segment" "branch_with_overrides" {
  name             = "Branch Office"
  starting_address = "10.10.0.0"
  ending_address   = "10.10.255.255"

  building             = "Branch HQ"
  department           = "Field Operations"
  override_buildings   = true
  override_departments = true
}
