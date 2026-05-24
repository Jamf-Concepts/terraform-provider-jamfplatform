# Active Directory binding. `password` is `WriteOnly` — sent to Jamf Pro
# on writes but never persisted in Terraform state. Bump `password_wo_version`
# to rotate the stored password on the next apply.
#
# TF attribute names mirror the Jamf Pro admin UI labels; the wire (XML
# element) names are documented in each attribute's schema description.
resource "jamfplatform_pro_directory_binding" "ad" {
  name                = "ad-prod"
  priority            = 1
  type                = "Active Directory"
  domain              = "corp.example.com"
  username            = "joiner-svc"
  password            = sensitive("change-me")
  password_wo_version = 1
  computer_ou         = "OU=Macs,DC=corp,DC=example,DC=com"

  active_directory = {
    create_mobile_account      = true
    require_confirmation       = true
    force_local_home_directory = true
    use_unc_path               = true
    network_protocol           = "smb"
    default_shell              = "/bin/bash"
    multiple_domains           = false
    preferred_domain           = "dc01.corp.example.com"
    admin_groups               = "Mac Admins,Domain Admins"
  }
}

# Apple Open Directory binding. Note: the admin UI labels this "Apple Open
# Directory" but the wire `type` value is the bare "Open Directory" —
# match the wire form.
resource "jamfplatform_pro_directory_binding" "open_directory" {
  name                = "od-staging"
  priority            = 2
  type                = "Open Directory"
  domain              = "ldap.staging.example.com"
  username            = "cn=joiner,dc=staging,dc=example,dc=com"
  password            = sensitive("change-me")
  password_wo_version = 1

  open_directory = {
    encrypt_using_ssl      = true
    perform_secure_bind    = true
    use_for_authentication = true
    use_for_contacts       = false
  }
}

# PowerBroker Identity Services. PowerBroker carries no per-type
# configuration on the wire, so no nested block is supplied — the `type`
# attribute on its own conveys the PowerBroker identity. The provider
# emits the empty <powerbroker_identity_services/> element automatically.
resource "jamfplatform_pro_directory_binding" "powerbroker" {
  name                = "pb-lab"
  priority            = 3
  type                = "PowerBroker Identity Services"
  domain              = "lab.example.com"
  username            = "joiner@lab.example.com"
  password            = sensitive("change-me")
  password_wo_version = 1
  computer_ou         = "OU=Macs,DC=lab,DC=example,DC=com"
}

# ADmitMac binding. `home_location` is the ADmitMac UI's "Home Location"
# field — distinct from the AD type's bool `force_local_home_directory`,
# even though both round-trip through a wire element named `local_home`.
resource "jamfplatform_pro_directory_binding" "admitmac" {
  name                = "admitmac-prod"
  priority            = 4
  type                = "ADmitMac"
  domain              = "corp.example.com"
  username            = "joiner-svc"
  password            = sensitive("change-me")
  password_wo_version = 1
  computer_ou         = "OU=Macs,DC=corp,DC=example,DC=com"

  admitmac = {
    require_confirmation = false
    home_location        = "Local"
    network_protocol     = "smb"
    default_shell        = "/bin/bash"
    mount_network_home   = false
    place_home_folders   = "/Users"
    admin_group          = "Mac Admins"
    cached_credentials   = 10
    add_user_to_local    = true
    users_ou             = "OU=Users,DC=corp,DC=example,DC=com"
    groups_ou            = "OU=Groups,DC=corp,DC=example,DC=com"
    printers_ou          = "OU=Printers,DC=corp,DC=example,DC=com"
    shared_folders_ou    = "OU=Shares,DC=corp,DC=example,DC=com"
  }
}

# Centrify binding. `update_pam` round-trips through the wire element
# <update_PAM> (uppercase preserved on the wire); the TF schema uses
# snake_case.
resource "jamfplatform_pro_directory_binding" "centrify" {
  name                = "centrify-prod"
  priority            = 5
  type                = "Centrify"
  domain              = "corp.example.com"
  username            = "joiner-svc"
  password            = sensitive("change-me")
  password_wo_version = 1

  centrify = {
    workstation_mode        = false
    overwrite_existing      = true
    update_pam              = true
    zone                    = "macs"
    preferred_domain_server = "dc01.corp.example.com"
  }
}
