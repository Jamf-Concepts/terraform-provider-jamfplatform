# Minimal printer using the bundled macOS Generic.ppd.
# use_generic defaults to true; the Jamf Pro server populates ppd_path with
# the system Generic.ppd path on its own.
resource "jamfplatform_pro_printer" "front_desk" {
  name = "Front Desk"
  uri  = "ipp://10.1.20.120/"
}

# Printer with an explicit PPD. use_generic must be false; ppd_path is the
# gate field: if omitted, the server falls back to Generic.ppd and silently
# flips use_generic back to true. ppd and ppd_contents are optional alongside.
resource "jamfplatform_pro_printer" "lab_color" {
  name        = "Lab Color"
  category    = "Printers"
  uri         = "ipp://printer.lab.example.com/queue1"
  cups_name   = "lab_color"
  location    = "Building 5, floor 2"
  model       = "HP DeskJet 2600 series"
  info        = "Drop sheets at the loading bay."
  notes       = "Created 2026 — replaces the LaserJet at desk 17."
  use_generic = false
  ppd         = "HP DeskJet 2600 series.ppd"
  ppd_path    = "/Library/Printers/PPDs/Contents/Resources/HP DeskJet 2600 series.ppd"
  # ppd_contents = file("${path.module}/HP DeskJet 2600 series.ppd")
  make_default    = true
  shared          = true
  os_requirements = "13.5, 16.0"
}
