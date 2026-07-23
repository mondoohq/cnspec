# Non-compliant: virtual node listener TLS mode is DISABLED, not STRICT/PERMISSIVE.
resource "aws_appmesh_virtual_node" "fail_example" {
  name      = "example-node"
  mesh_name = "example-mesh"

  spec {
    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        mode = "DISABLED"

        certificate {
          acm {
            certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/example"
          }
        }
      }
    }
  }
}
