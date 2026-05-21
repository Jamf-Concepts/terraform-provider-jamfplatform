resource "jamfplatform_pro_building" "hq" {
  name             = "HQ"
  city             = "Minneapolis"
  country          = "USA"
  state_province   = "MN"
  street_address_1 = "100 Washington Ave S"
  street_address_2 = "Suite 1100"
  zip_postal_code  = "55401"
}

resource "jamfplatform_pro_building" "branch" {
  name = "Eau Claire Office"
  city = "Eau Claire"
}
