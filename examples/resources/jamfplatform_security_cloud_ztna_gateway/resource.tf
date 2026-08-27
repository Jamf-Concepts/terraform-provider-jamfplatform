# A gateway takes one of two forms, chosen by whether the ipsec block is present.

# Form 1 — dedicated internet gateway. Omit ipsec, and Jamf provisions a pair of
# private egress IP addresses, reported in dedicated_egress_ip_addresses.
resource "jamfplatform_security_cloud_ztna_gateway" "internet" {
  name       = "London Internet Egress"
  datacenter = "eu-west-2"
  tenant_ids = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }
}

# Form 2 — dedicated IPSec gateway. Set ipsec to build a tunnel to your own VPN
# concentrator. Adding or removing the block later replaces the gateway: Jamf
# Security Cloud will not convert one form into the other.
resource "jamfplatform_security_cloud_ztna_gateway" "ipsec" {
  name       = "Frankfurt Private Apps"
  datacenter = "eu-central-1"
  tenant_ids = [var.security_cloud_tenant_id]

  contact = {
    name  = "Network Operations"
    email = "netops@example.com"
  }

  # The source addresses your firewall must allow. The admin UI lists the
  # addresses each egress region offers when you pick the region; supply both for
  # dynamic addressing, or one to pin a single source.
  availability_zones = ["3.66.107.208", "3.121.43.105"]

  ipsec = {
    key_exchange = "ikev2"

    # "Phase 1" and "Phase 2" in the admin UI. The Diffie-Hellman group values
    # appear there alongside the group number — "Group 14 (modp2048)".
    ike = {
      encryption       = "aes256"
      integrity        = "sha512"
      dh_group         = "modp2048"
      lifetime_seconds = 28800
    }

    esp = {
      encryption       = "aes256"
      integrity        = "sha512"
      dh_group         = "modp2048"
      lifetime_seconds = 28800
    }

    jamf_side = {
      host   = "%any"
      ike_id = "wpa.wandera.com"

      # Exactly one private range, and it must not exist anywhere else on your
      # network: 10.0.0.0/8 (/8-/30), 172.16.0.0/12 (/12-/30) or
      # 192.168.0.0/16 (/16-/30).
      subnet = "172.16.0.0/12"

      # Never stored in Terraform state. Bump shared_secret_wo_version to rotate
      # it; leaving the version alone keeps the key Jamf already has.
      shared_secret            = var.ipsec_shared_secret
      shared_secret_wo_version = 1
    }

    customer_side = {
      host    = "203.0.113.10"
      ike_id  = "vpn.example.com"
      subnets = ["10.100.0.0/16", "10.101.0.0/16"]

      # Case-sensitive. See the schema for the full list.
      vendor = "strongSwan"
    }
  }
}

variable "security_cloud_tenant_id" {
  description = "Tenant granted access to these gateways."
  type        = string
}

variable "ipsec_shared_secret" {
  description = "IPSec pre-shared key."
  type        = string
  sensitive   = true
}
