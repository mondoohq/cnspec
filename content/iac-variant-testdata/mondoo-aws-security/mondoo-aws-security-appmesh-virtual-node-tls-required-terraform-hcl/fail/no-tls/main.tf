# Non-compliant: virtual node listener has no TLS configuration.
resource "aws_appmesh_virtual_node" "fail_example" {
  name      = "example-node"
  mesh_name = "example-mesh"

  spec {
    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }
    }
  }
}
