# Non-compliant: client-broker traffic allows PLAINTEXT instead of TLS.
resource "aws_msk_cluster" "fail_example" {
  cluster_name = "example"

  encryption_info {
    encryption_in_transit {
      client_broker = "PLAINTEXT"
    }
  }
}
