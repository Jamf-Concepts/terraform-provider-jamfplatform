data "jamfplatform_pro_buildings" "minneapolis" {
  filter = [
    {
      selector = "city"
      argument = "Minneapolis"
    }
  ]
}

data "jamfplatform_pro_buildings" "by_name_prefix" {
  filter = [
    {
      selector = "name"
      argument = "HQ*"
    }
  ]
}

output "minneapolis_buildings" {
  value = data.jamfplatform_pro_buildings.minneapolis.buildings
}
