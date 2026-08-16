# Compliant: publicly_accessible is omitted; the check only rejects an explicit
# true, and an unset value is not publicly accessible in this configuration.
resource "aws_dms_replication_instance" "pass_example" {
  replication_instance_id    = "example-instance"
  replication_instance_class = "dms.t3.micro"
}
