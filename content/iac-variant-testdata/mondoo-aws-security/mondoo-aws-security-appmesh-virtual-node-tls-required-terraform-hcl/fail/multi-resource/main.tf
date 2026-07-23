# Non-compliant: one of two virtual nodes has a listener without TLS.
resource "aws_appmesh_virtual_node" "ok" {
  name      = "secure-node"
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
  }
}

resource "aws_appmesh_virtual_node" "bad" {
  name      = "insecure-node"
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
