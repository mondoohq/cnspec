# Non-compliant: storage encryption is disabled, so there is no CMK either.
resource "aws_db_instance" "fail_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
  storage_encrypted = false
}
