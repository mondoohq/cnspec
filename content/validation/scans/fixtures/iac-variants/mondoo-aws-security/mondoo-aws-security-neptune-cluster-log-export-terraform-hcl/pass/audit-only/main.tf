# Compliant: audit logs are exported to CloudWatch.
resource "aws_neptune_cluster" "pass_example" {
  cluster_identifier                   = "example"
  enable_cloudwatch_logs_exports       = ["audit"]
}
