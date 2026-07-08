# Non-compliant: DocumentDB cluster is encrypted but relies on the default AWS-managed key.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
  storage_encrypted  = true
}
