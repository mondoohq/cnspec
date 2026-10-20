# Non-compliant: two block device mappings; one EBS volume is not encrypted.
resource "aws_launch_template" "fail_example" {
  name = "fail_example_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
    }
  }

  block_device_mappings {
    device_name = "/dev/xvdb"
    ebs {
      volume_size = 50
      encrypted   = false
    }
  }
}

resource "aws_eks_node_group" "fail_example" {
  cluster_name  = "example"
  node_role_arn = var.node_role_arn
  subnet_ids    = var.subnet_ids

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  launch_template {
    id      = aws_launch_template.fail_example.id
    version = aws_launch_template.fail_example.latest_version
  }
}
