# Non-compliant: deletion_protection is omitted, so it defaults to disabled.
resource "aws_db_instance" "fail_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
}
