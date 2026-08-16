# Non-compliant: DocumentDB cluster omits storage_encrypted, which defaults to
# unencrypted storage.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "example-cluster"
  master_username    = "admin"
  master_password    = "SuperSecretPassw0rd"
}
