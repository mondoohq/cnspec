# Indexed content and conversations use the AWS owned default key.
resource "aws_qbusiness_application" "example" {
  display_name                 = "enterprise-assistant"
  iam_service_role_arn         = "arn:aws:iam::111122223333:role/qbusiness"
  identity_center_instance_arn = "arn:aws:sso:::instance/ssoins-0123456789abcdef"

  attachments_configuration {
    attachments_control_mode = "ENABLED"
  }
}
