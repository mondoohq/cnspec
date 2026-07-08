# Compliant: DocumentDB cluster attaches a dedicated VPC security group by reference.
resource "aws_security_group" "docdb" {
  name        = "docdb-access"
  description = "Access to DocumentDB"
  vpc_id      = "vpc-0123456789abcdef0"
}

resource "aws_docdb_cluster" "pass_example" {
  cluster_identifier     = "pass-example"
  master_username        = "admin"
  master_password        = "mustbeeightchars"
  vpc_security_group_ids = [aws_security_group.docdb.id]
}
