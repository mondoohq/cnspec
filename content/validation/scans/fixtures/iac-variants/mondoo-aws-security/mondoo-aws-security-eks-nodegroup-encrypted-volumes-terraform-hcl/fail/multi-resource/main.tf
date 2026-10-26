# Non-compliant: two launch templates; exactly one has an unencrypted EBS
# volume. .all() over resources must still fail.
resource "aws_launch_template" "good" {
  name = "good_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = true
    }
  }
}

resource "aws_launch_template" "bad" {
  name = "bad_template"

  block_device_mappings {
    device_name = "/dev/xvda"
    ebs {
      volume_size = 20
      encrypted   = false
    }
  }
}

resource "aws_eks_node_group" "good" {
  cluster_name  = "example"
  node_role_arn = var.node_role_arn
  subnet_ids    = var.subnet_ids

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  launch_template {
    id      = aws_launch_template.good.id
    version = aws_launch_template.good.latest_version
  }
}

resource "aws_eks_node_group" "bad" {
  cluster_name  = "example"
  node_role_arn = var.node_role_arn
  subnet_ids    = var.subnet_ids

  scaling_config {
    desired_size = 1
    max_size     = 1
    min_size     = 1
  }

  launch_template {
    id      = aws_launch_template.bad.id
    version = aws_launch_template.bad.latest_version
  }
}
