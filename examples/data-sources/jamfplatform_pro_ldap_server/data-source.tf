# Look up an LDAP server by its numeric ID.
data "jamfplatform_pro_ldap_server" "by_id" {
  id = "31"
}

# Or by its exact display name.
data "jamfplatform_pro_ldap_server" "by_name" {
  name = "Corporate AD"
}

output "ldap_hostname" {
  value = data.jamfplatform_pro_ldap_server.by_name.connection_settings.hostname
}
