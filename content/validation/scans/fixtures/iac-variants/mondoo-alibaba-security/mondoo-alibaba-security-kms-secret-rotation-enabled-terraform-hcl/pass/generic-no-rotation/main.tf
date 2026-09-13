# Alibaba Cloud KMS cannot rotate a generic secret automatically, so the check
# does not hold it to a setting it cannot enable.
resource "alicloud_kms_secret" "api_token" {
  secret_name               = "api-token"
  secret_type               = "Generic"
  secret_data               = var.api_token
  version_id                = "v1"
  enable_automatic_rotation = false
}
