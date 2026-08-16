# Non-compliant: DocumentDB cluster leaves storage encryption disabled, so there
# is neither encryption nor a customer-managed key.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
  storage_encrypted  = false
}
