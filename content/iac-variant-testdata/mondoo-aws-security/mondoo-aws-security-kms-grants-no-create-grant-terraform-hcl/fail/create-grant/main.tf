# Non-compliant: grant allows the CreateGrant operation.
resource "aws_kms_grant" "fail_example" {
  name              = "fail-example"
  key_id            = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
  grantee_principal = "arn:aws:iam::123456789012:role/example"
  operations        = ["Encrypt", "Decrypt", "CreateGrant"]
}
