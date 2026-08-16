# Compliant: session_duration is exactly 12 hours (the allowed maximum).
resource "aws_ssoadmin_permission_set" "pass_example" {
  name             = "admin"
  instance_arn     = "arn:aws:sso:::instance/ssoins-example"
  session_duration = "PT12H"
}
