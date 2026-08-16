# Non-compliant: DocumentDB cluster does not configure any CloudWatch log exports.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "fail-example"
  master_username    = "admin"
  master_password    = "mustbeeightchars"
}
