# Non-compliant: DocumentDB cluster has no explicit VPC security groups.
resource "aws_docdb_cluster" "fail_example" {
  cluster_identifier     = "fail-example"
  master_username        = "admin"
  master_password        = "mustbeeightchars"
  vpc_security_group_ids = []
}
