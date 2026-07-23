# Non-compliant: listeners generated from a variable via a dynamic block, none
# of which configure TLS.
variable "listeners" {
  type    = list(number)
  default = [8080, 9090]
}

resource "aws_appmesh_virtual_node" "fail_dynamic" {
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
      }
    }
  }
}
