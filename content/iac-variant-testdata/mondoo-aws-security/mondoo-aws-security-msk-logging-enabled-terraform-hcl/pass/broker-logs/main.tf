# Compliant: logging_info defines broker_logs.
resource "aws_msk_cluster" "pass_example" {
  cluster_name = "example"

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = "example-log-group"
      }
    }
  }
}
