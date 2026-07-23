# Compliant: virtual node listener enforces TLS in PERMISSIVE mode.
resource "aws_appmesh_virtual_node" "pass_example" {
  name      = "example-node"
  mesh_name = "example-mesh"

  spec {
    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        mode = "PERMISSIVE"

        certificate {
          acm {
            certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/example"
          }
        }
      }
    }
  }
}
