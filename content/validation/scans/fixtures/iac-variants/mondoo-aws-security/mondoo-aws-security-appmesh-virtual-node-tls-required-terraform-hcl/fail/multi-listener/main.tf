# Non-compliant: node has two listeners; the second one has no TLS config.
resource "aws_appmesh_virtual_node" "fail_multi_listener" {
  name      = "example-node"
  mesh_name = "example-mesh"

  spec {
    listener {
      port_mapping {
        port     = 8080
        protocol = "http"
      }

      tls {
        mode = "STRICT"

        certificate {
          acm {
            certificate_arn = "arn:aws:acm:us-east-1:123456789012:certificate/example"
          }
        }
      }
    }

    listener {
      port_mapping {
        port     = 9090
        protocol = "http"
      }
    }
  }
}
