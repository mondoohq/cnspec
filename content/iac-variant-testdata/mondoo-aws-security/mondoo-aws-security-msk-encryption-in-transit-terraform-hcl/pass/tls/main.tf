# Compliant: encryption in transit enforces TLS for client-broker traffic.
resource "aws_msk_cluster" "pass_example" {
  cluster_name = "example"

  encryption_info {
    encryption_in_transit {
      client_broker = "TLS"
    }
  }
}
