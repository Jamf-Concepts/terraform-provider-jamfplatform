# Manage an on-premises Active Directory LDAP server with an authenticated
# (simple) bind. This resource is for classic, directly-reachable directories;
# cloud directories (Google, Microsoft Entra) use
# jamfplatform_pro_cloud_identity_provider instead.
resource "jamfplatform_pro_ldap_server" "corp_ad" {
  connection_settings = {
    display_name        = "Corporate AD"
    directory_service   = "Active Directory" # Active Directory | Open Directory | eDirectory | Custom
    hostname            = "ldap.corp.example.com"
    port                = 636
    use_ssl             = true
    authentication_type = "simple" # none | simple | CRAM-MD5 | DIGEST-MD5

    account = {
      distinguished_username = "CN=jamf-bind,CN=Users,DC=corp,DC=example,DC=com"
      # password is WriteOnly: sent to Jamf Pro but never stored in state.
      # Bump password_wo_version to rotate the stored password.
      password            = var.ldap_bind_password
      password_wo_version = 1
    }

    connection_timeout = 15
    search_timeout     = 60
    referral_response  = "" # "" (use LDAP default) | follow | ignore
    use_wildcards      = true
  }

  mappings_for_users = {
    user_mappings = {
      object_class_limitation = "any"
      object_classes          = "organizationalPerson"
      search_base             = "OU=Users,DC=corp,DC=example,DC=com"
      search_scope            = "All Subtrees" # All Subtrees | First Level Only
      user_id                 = "uSNCreated"
      username                = "mail"
      real_name               = "displayName"
      email_address           = "mail"
      user_uuid               = "objectGUID"
    }

    user_group_mappings = {
      object_class_limitation = "any"
      object_classes          = "group"
      search_base             = "OU=Groups,DC=corp,DC=example,DC=com"
      search_scope            = "All Subtrees"
      group_id                = "uSNCreated"
      group_name              = "sAMAccountName"
      group_uuid              = "objectGUID"
    }

    user_group_membership_mappings = {
      membership_location = "group object" # group object | user object | Other
      member_user_mapping = "member"
      use_dn              = true
      use_ldap_compare    = true
      recursive_lookups   = true
    }
  }
}

variable "ldap_bind_password" {
  type      = string
  sensitive = true
}

# Anonymous (unauthenticated) bind: omit the account block entirely.
resource "jamfplatform_pro_ldap_server" "anon" {
  connection_settings = {
    display_name        = "Read-only Directory"
    directory_service   = "Open Directory"
    hostname            = "ldap.example.org"
    port                = 389
    use_ssl             = false
    authentication_type = "none"
  }
}
