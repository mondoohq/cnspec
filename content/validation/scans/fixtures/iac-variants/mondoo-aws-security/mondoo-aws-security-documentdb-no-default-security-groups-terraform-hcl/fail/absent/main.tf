# Non-compliant: DocumentDB cluster does not attach any VPC security groups,
# so it falls back to the default security group.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier = "fail-example"
  master_username    = "admin"
  master_password    = "mustbeeightchars"
}
