# Automatic rotation does not apply to asymmetric signing keys, so the check
# scopes itself to ENCRYPT/DECRYPT keys and this one is out of scope.
resource "alicloud_kms_key" "signing" {
  description        = "Artifact signing key"
  key_usage          = "SIGN/VERIFY"
  key_spec           = "RSA_2048"
  automatic_rotation = "Disabled"
}
