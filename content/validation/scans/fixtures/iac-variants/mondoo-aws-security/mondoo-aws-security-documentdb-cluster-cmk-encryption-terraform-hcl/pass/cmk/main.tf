# Compliant: DocumentDB cluster is encrypted with a customer-managed KMS key.
resource "aws_docdb_cluster" "pass_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
  storage_encrypted  = true
  kms_key_id         = "arn:aws:kms:us-east-1:123456789012:key/abcd1234-a123-456a-a12b-a123b4cd56ef"
}
