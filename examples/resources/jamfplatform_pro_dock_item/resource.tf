resource "jamfplatform_pro_dock_item" "calculator" {
  name = "Calculator"
  type = "App"
  path = "/Applications/Calculator.app"
}

resource "jamfplatform_pro_dock_item" "readme" {
  name = "Readme"
  type = "File"
  path = "file://localhost/Library/Documentation/README.txt"
}

resource "jamfplatform_pro_dock_item" "downloads" {
  name = "Downloads"
  type = "Folder"
  path = "~/Downloads"
}
