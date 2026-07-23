# Non-compliant: MSK cluster encryption_info has no KMS key ARN.
resource "aws_msk_cluster" "fail_example" {
  cluster_name = "fail-example"

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
    }
  }
}
