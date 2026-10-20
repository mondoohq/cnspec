# Compliant: launch template block device mappings generated via a dynamic
# block; the EBS volume is encrypted.
variable "mappings" {
  type = list(object({ device_name = string, size = number }))
  default = [
    { device_name = "/dev/xvda", size = 20 },
  ]
}

resource "aws_launch_template" "pass_example" {
  name = "pass_example_template"

  dynamic "block_device_mappings" {
    for_each = var.mappings
    content {
      device_name = block_device_mappings.value.device_name
      ebs {
        volume_size = block_device_mappings.value.size
        encrypted   = true
      }
    }
  }
}

resource "aws_eks_node_group" "pass_example" {
  cluster_name  = "example"
  node_role_arn = var.node_role_arn
  subnet_ids    = var.subnet_ids

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  launch_template {
    id      = aws_launch_template.pass_example.id
    version = aws_launch_template.pass_example.latest_version
  }
}
