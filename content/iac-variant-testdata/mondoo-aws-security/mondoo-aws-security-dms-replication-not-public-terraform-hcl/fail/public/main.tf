# Non-compliant: DMS replication instance is publicly accessible.
resource "aws_dms_replication_instance" "fail_example" {
  replication_instance_id    = "example-instance"
  replication_instance_class = "dms.t3.micro"
  publicly_accessible        = true
}
