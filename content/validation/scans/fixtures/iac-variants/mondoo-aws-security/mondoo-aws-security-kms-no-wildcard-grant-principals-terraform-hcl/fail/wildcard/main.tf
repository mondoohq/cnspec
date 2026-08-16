# Non-compliant: KMS grant targets the wildcard principal "*".
resource "aws_kms_grant" "fail_example" {
  name              = "fail-grant"
  key_id            = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  grantee_principal = "*"
  operations        = ["Encrypt", "Decrypt"]
}
