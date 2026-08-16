# Non-compliant: no logging_info block, so broker logging is not configured.
resource "aws_msk_cluster" "fail_example" {
  cluster_name = "example"
}
