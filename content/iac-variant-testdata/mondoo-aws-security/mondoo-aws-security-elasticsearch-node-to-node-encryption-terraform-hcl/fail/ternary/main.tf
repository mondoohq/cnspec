# Non-compliant: node-to-node encryption toggled by a ternary with a false active branch.
variable "encrypt" {
  type    = bool
  default = false
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  node_to_node_encryption {
    enabled = var.encrypt ? true : false
  }
}
