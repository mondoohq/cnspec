# Compliant: broker logs are shipped to S3 instead of CloudWatch.
resource "aws_msk_cluster" "pass_example" {
  cluster_name = "example"

  logging_info {
    broker_logs {
      s3 {
        enabled = true
        bucket  = "example-msk-logs"
        prefix  = "logs/"
      }
    }
  }
}
