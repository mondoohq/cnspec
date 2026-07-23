# Non-compliant: audit logs are not among the exported log types.
resource "aws_neptune_cluster" "fail_example" {
  cluster_identifier             = "example"
  enable_cloudwatch_logs_exports = ["slowquery"]
}
