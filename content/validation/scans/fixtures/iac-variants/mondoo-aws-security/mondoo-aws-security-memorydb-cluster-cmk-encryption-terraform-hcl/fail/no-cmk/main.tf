# Non-compliant: no kms_key_arn, relies on default encryption.
resource "aws_memorydb_cluster" "fail_example" {
  name      = "example"
  node_type = "db.t4g.small"
  acl_name  = "open-access"
}
