# Non-compliant: TLS_PLAINTEXT still permits unencrypted client-broker traffic.
resource "aws_msk_cluster" "fail_example" {
  cluster_name = "example"

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS_PLAINTEXT"
    }
  }
}
