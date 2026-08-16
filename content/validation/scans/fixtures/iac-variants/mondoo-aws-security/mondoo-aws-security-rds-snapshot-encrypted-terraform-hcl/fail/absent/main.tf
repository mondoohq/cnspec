# Realistic aws_db_snapshot: "encrypted" is a read-only exported attribute and is not
# set in configuration; encryption is inherited from the source DB instance.
resource "aws_db_snapshot" "example" {
  db_instance_identifier = "example"
  db_snapshot_identifier = "example-snapshot"
}
