# Non-compliant: no encryption settings, so storage is unencrypted and has no CMK.
resource "aws_db_instance" "fail_example" {
  identifier        = "example"
  engine            = "mysql"
  instance_class    = "db.t3.micro"
  allocated_storage = 20
}
