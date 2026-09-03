# Jamf Cloud Distribution Service (JCDS). No credentials required.
resource "jamfplatform_pro_cloud_distribution_point" "this" {
  cdn_type = "JAMF_CLOUD"
  master   = false
}

# Amazon Web Services (S3 + CloudFront signed URLs). Credentials required.
# username = Access Key ID, password = Secret access key (WriteOnly).
# private_key is the CloudFront signing key, required when require_signed_urls = true.
#
# resource "jamfplatform_pro_cloud_distribution_point" "aws" {
#   cdn_type            = "AMAZON_S3"
#   username            = "AKIAEXAMPLE"
#   password            = var.s3_secret_access_key # WriteOnly
#   require_signed_urls = true
#   key_pair_id         = "K1R8C5EXAMPLE"
#   private_key         = filebase64("${path.module}/cloudfront-key.pem") # WriteOnly
#   expiration_seconds  = 3600
# }

# Akamai: username/password plus upload, directory, and download endpoints.
#
# resource "jamfplatform_pro_cloud_distribution_point" "akamai" {
#   cdn_type     = "AKAMAI"
#   username     = "akamai-user"
#   password     = var.akamai_password # WriteOnly
#   upload_url   = "ftp://mycompany.upload.akamai.com"
#   directory    = "123456"
#   download_url = "https://download.mycompany.com"
# }
