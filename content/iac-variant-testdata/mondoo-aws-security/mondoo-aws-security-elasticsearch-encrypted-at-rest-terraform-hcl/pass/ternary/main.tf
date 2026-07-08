# Compliant: encryption toggled by a ternary whose active branch is true.
variable "encrypt" {
  type    = bool
  default = true
}

resource "aws_elasticsearch_domain" "compliant" {
  domain_name           = "compliant"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = var.encrypt ? true : false
  }
}
