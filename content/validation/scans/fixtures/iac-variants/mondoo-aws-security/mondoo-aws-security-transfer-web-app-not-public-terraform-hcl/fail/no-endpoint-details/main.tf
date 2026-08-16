resource "aws_transfer_web_app" "example" {
  access_endpoint = "https://transfer.example.com"

  identity_provider_details {
    identity_center_config {
      instance_arn = aws_ssoadmin_instance.example.arn
    }
  }
}
