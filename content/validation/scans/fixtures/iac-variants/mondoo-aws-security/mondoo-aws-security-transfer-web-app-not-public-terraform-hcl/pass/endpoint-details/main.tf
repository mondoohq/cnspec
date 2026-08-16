resource "aws_transfer_web_app" "example" {
  identity_provider_details {
    identity_center_config {
      instance_arn = aws_ssoadmin_instance.example.arn
    }
  }

  endpoint_details {
    vpc {
      vpc_id             = aws_vpc.main.id
      subnet_ids         = [aws_subnet.main.id]
      security_group_ids = [aws_security_group.web.id]
    }
  }
}
