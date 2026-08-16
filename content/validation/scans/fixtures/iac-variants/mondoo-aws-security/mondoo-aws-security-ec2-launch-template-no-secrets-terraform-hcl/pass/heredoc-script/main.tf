resource "aws_launch_template" "compliant" {
  name          = "example"
  image_id      = "ami-12345678"
  instance_type = "t3.micro"
  user_data     = <<-EOT
    #!/bin/bash
    yum update -y
    systemctl enable nginx
  EOT
}
