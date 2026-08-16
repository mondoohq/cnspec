# Compliant: auto_minor_version_upgrade is omitted, and both the AWS API and the
# Terraform provider default it to true, so the instance still receives patches.
resource "aws_db_instance" "pass_default" {
  identifier        = "pass-default"
  engine            = "postgres"
  instance_class    = "db.t3.medium"
  allocated_storage = 20
}
