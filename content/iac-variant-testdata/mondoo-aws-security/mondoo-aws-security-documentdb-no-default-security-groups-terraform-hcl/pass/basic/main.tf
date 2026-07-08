# Compliant: DocumentDB cluster attaches explicit VPC security groups.
resource "aws_docdb_cluster" "pass_example" {
  cluster_identifier     = "pass-example"
  master_username        = "admin"
  master_password        = "mustbeeightchars"
  vpc_security_group_ids = ["sg-0123456789abcdef0"]
}
