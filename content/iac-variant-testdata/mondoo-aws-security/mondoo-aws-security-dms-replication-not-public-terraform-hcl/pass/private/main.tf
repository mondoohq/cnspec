# Compliant: DMS replication instance is not publicly accessible.
resource "aws_dms_replication_instance" "pass_example" {
  replication_instance_id    = "example-instance"
  replication_instance_class = "dms.t3.micro"
  publicly_accessible        = false
}
