# Compliant: DocumentDB cluster enables storage encryption.
resource "aws_docdb_cluster" "pass_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
  storage_encrypted  = true
}
