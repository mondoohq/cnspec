# Non-compliant: iam_database_authentication_enabled is omitted, so it defaults to disabled.
resource "aws_db_instance" "fail_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
}
