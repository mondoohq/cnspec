# Compliant: KMS grant targets a specific principal, not "*".
resource "aws_kms_grant" "pass_example" {
  name              = "pass-grant"
  key_id            = "arn:aws:kms:us-east-1:111122223333:key/abcd"
  grantee_principal = "arn:aws:iam::111122223333:role/app"
  operations        = ["Encrypt", "Decrypt"]
}
