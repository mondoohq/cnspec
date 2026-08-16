# Non-compliant: a counted mesh allows all outbound egress traffic.
resource "aws_appmesh_mesh" "fail_count" {
  count = 2
  name  = "example-mesh-${count.index}"

  spec {
    egress_filter {
      type = "ALLOW_ALL"
    }
  }
}
