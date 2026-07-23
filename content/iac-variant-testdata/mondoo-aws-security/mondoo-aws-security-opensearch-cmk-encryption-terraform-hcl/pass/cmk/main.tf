resource "aws_opensearch_domain" "example" {
  domain_name = "example"

  encrypt_at_rest {
    enabled    = true
    kms_key_id = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
  }
}
