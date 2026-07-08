# Non-compliant: one of two meshes allows all outbound egress traffic.
resource "aws_appmesh_mesh" "ok" {
  name = "safe-mesh"

  spec {
    egress_filter {
      type = "DROP_ALL"
    }
  }
}

resource "aws_appmesh_mesh" "bad" {
  name = "open-mesh"

  spec {
    egress_filter {
      type = "ALLOW_ALL"
    }
  }
}
