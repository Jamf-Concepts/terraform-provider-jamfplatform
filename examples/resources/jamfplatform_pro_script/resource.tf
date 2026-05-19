resource "jamfplatform_pro_script" "rotate_local_admin_password" {
  name            = "Rotate Local Admin Password"
  priority        = "AFTER"
  info            = "Rotates the local admin password on macOS clients."
  notes           = "Owned by Endpoint Engineering. See runbook in confluence."
  os_requirements = "13.0.x,14.0.x,15.0.x"

  parameter_4 = "Target Username"
  parameter_5 = "Password Length"

  script_contents = <<-EOT
    #!/bin/sh
    set -eu
    USER="$${4:-localadmin}"
    LENGTH="$${5:-24}"
    NEW_PASSWORD=$(/usr/bin/openssl rand -base64 "$LENGTH" | tr -dc 'A-Za-z0-9' | cut -c1-"$LENGTH")
    /usr/bin/dscl . -passwd "/Users/$USER" "$NEW_PASSWORD"
    echo "Rotated password for $USER"
  EOT
}

resource "jamfplatform_pro_script" "cleanup_temp" {
  name            = "Cleanup /tmp"
  priority        = "AT_REBOOT"
  script_contents = <<-EOT
    #!/bin/sh
    /bin/rm -rf /tmp/*
  EOT
}
