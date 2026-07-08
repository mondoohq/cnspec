# Non-compliant: no CloudWatch log exports are configured, so audit logs are not exported.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier = "example"
  engine             = "neptune"
}
