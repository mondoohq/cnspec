# Compliant: DB instance has IAM database authentication enabled.
resource "aws_db_instance" "pass_example" {
  identifier                          = "example"
  engine                              = "mysql"
  instance_class                      = "db.t3.micro"
  allocated_storage                   = 20
  iam_database_authentication_enabled = true
}
