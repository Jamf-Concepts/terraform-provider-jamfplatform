# List Jamf Pro packages with an RSQL filter against the V1 /packages
# endpoint. The server enforces a strict 8-selector whitelist:
#   packageName, fileName, id, categoryId, info, notes,
#   manifestFileName, cloudTransferStatus.
# Filtering on any other field returns HTTP 400 from Jamf Pro — the
# provider rejects unsupported selectors at plan time via the schema.

list "jamfplatform_pro_package" "by_name_prefix" {
  provider = jamfplatform

  config {
    filter = [
      {
        selector = "packageName"
        operator = "=="
        # RSQL "==" with a trailing "*" matches packages whose name is
        # exactly that literal — wildcards do NOT expand. Use the value
        # you mean to match.
        argument = "MyApp 1.0.0"
      }
    ]
  }
}

list "jamfplatform_pro_package" "ready_only" {
  provider = jamfplatform

  config {
    filter = [
      {
        selector = "cloudTransferStatus"
        operator = "=="
        argument = "READY"
      }
    ]
  }
}
