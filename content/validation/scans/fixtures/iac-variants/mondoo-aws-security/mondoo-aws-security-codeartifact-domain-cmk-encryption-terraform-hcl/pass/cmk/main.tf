# Compliant: domain encrypted with a customer managed KMS key.
resource "aws_codeartifact_domain" "pass_example" {
  domain         = "example-domain"
  encryption_key = "arn:aws:kms:us-east-1:111122223333:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
}
