# Compliant: listeners are generated from a variable via a dynamic block, and
# every generated listener enforces STRICT TLS. Generating listeners from a
# variable is a common Terraform idiom for multi-port services.
variable "listeners" {
  type    = list(number)
  default = [8080, 9090]
}

resource "aws_appmesh_virtual_node" "pass_dynamic" {
  name      = "example-node"
  mesh_name = "example-mesh"

  spec {
    dynamic "listener" {
      for_each = var.listeners
      content {
        port_mapping {
          port     = listener.value
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
}
