# Compliant: no aws_ssoadmin_permission_set_inline_policy resources present.
resource "aws_ssoadmin_permission_set" "pass_example" {
  name         = "example"
  instance_arn = "arn:aws:sso:::instance/ssoins-example"
}
