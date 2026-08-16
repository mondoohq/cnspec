# Compliant: grant does not include the CreateGrant operation.
resource "aws_kms_grant" "pass_example" {
  name              = "pass-example"
  key_id            = "arn:aws:kms:us-east-1:123456789012:key/abcd-1234"
  grantee_principal = "arn:aws:iam::123456789012:role/example"
  operations        = ["Encrypt", "Decrypt", "GenerateDataKey"]
}
