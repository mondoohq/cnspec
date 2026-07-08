# Non-compliant: mesh egress filter allows all outbound traffic.
resource "aws_appmesh_mesh" "fail_example" {
  name = "example-mesh"

  spec {
    egress_filter {
      type = "ALLOW_ALL"
    }
  }
}
