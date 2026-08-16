# Compliant: snapshot is encrypted.
resource "aws_db_snapshot" "example" {
  db_instance_identifier = "example"
  db_snapshot_identifier = "example-snapshot"
  encrypted              = true
}
