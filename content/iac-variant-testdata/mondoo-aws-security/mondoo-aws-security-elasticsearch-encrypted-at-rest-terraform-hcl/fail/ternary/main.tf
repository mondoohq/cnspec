# Non-compliant: encryption toggled by a ternary whose active branch is false.
variable "encrypt" {
  type    = bool
  default = false
}

resource "aws_elasticsearch_domain" "violating" {
  domain_name           = "violating"
  elasticsearch_version = "7.10"

  encrypt_at_rest {
    enabled = var.encrypt ? true : false
  }
}
