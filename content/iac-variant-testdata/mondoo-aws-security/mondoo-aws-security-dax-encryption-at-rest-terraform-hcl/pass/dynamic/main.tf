# Compliant: server-side encryption enabled via a conditional dynamic block.
variable "encrypt" {
  type    = bool
  default = true
}

resource "aws_dax_cluster" "pass_dynamic" {
  cluster_name       = "pass-dynamic"
  iam_role_arn       = "arn:aws:iam::123456789012:role/dax"
  node_type          = "dax.r4.large"
  replication_factor = 1

  dynamic "server_side_encryption" {
    for_each = var.encrypt ? [1] : []
    content {
      enabled = true
    }
  }
}
