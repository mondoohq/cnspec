# Indexed content and conversations are encrypted under a customer managed key.
resource "aws_qbusiness_application" "example" {
  display_name                 = "enterprise-assistant"
  iam_service_role_arn         = "arn:aws:iam::111122223333:role/qbusiness"
  identity_center_instance_arn = "arn:aws:sso:::instance/ssoins-0123456789abcdef"

  attachments_configuration {
    attachments_control_mode = "ENABLED"
  }

  encryption_configuration {
    kms_key_id = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"
  }
}
