# Compliant: database is not publicly accessible.
resource "aws_lightsail_database" "pass_example" {
  relational_database_name = "example"
  availability_zone        = "us-east-1a"
  master_database_name     = "exampledb"
  master_password          = "example-password-123"
  master_username          = "admin"
  blueprint_id             = "mysql_8_0"
  bundle_id                = "micro_1_0"
  publicly_accessible      = false
}
