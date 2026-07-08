# Non-compliant: DB instance has IAM database authentication disabled.
resource "aws_db_instance" "fail_example" {
  identifier                          = "example"
  engine                              = "mysql"
  instance_class                      = "db.t3.micro"
  allocated_storage                   = 20
  iam_database_authentication_enabled = false
}
