# Non-compliant: DMS replication instance relies on the default AWS-managed key.
resource "aws_dms_replication_instance" "fail_example" {
  replication_instance_id    = "example-instance"
  replication_instance_class = "dms.t3.micro"
}
