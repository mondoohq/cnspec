# Non-compliant: DocumentDB cluster disables storage encryption.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
  storage_encrypted  = false
}
