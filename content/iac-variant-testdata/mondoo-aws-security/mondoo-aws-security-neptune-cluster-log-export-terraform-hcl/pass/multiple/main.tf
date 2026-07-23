# Compliant: audit is included among several exported CloudWatch log types.
resource "aws_neptune_cluster" "pass_example" {
  cluster_identifier             = "example"
  enable_cloudwatch_logs_exports = ["audit", "slowquery"]
}
