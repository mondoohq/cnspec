# Compliant: launch template EBS volume is encrypted.
resource "aws_launch_template" "pass_example" {
  name = "pass_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
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
