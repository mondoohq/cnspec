# An omitted secret_type creates a generic secret, which cannot rotate
# automatically and is out of scope.
resource "alicloud_kms_secret" "api_token" {
  secret_name = "api-token"
  secret_data = var.api_token
  version_id  = "v1"
}
