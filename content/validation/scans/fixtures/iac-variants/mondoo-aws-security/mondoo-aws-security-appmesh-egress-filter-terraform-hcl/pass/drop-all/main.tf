# Compliant: mesh egress filter drops all unmatched outbound traffic.
resource "aws_appmesh_mesh" "pass_example" {
  name = "example-mesh"

  spec {
    egress_filter {
      type = "DROP_ALL"
    }
  }
}
