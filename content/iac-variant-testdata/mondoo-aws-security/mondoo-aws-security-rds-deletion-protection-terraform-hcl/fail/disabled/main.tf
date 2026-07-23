# Non-compliant: DB instance has deletion protection disabled.
resource "aws_db_instance" "fail_example" {
  identifier          = "example"
  engine              = "mysql"
  instance_class      = "db.t3.micro"
  allocated_storage   = 20
  deletion_protection = false
}
