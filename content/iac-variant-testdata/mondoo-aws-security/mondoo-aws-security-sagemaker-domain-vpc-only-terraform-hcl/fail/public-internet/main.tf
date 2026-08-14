# PublicInternetOnly routes notebook traffic through an AWS-managed internet
# gateway, bypassing the VPC's egress controls and inspection.
resource "aws_sagemaker_domain" "research" {
  domain_name             = "research"
  auth_mode               = "IAM"
  vpc_id                  = aws_vpc.prod.id
  subnet_ids              = [aws_subnet.private_a.id]
  app_network_access_type = "PublicInternetOnly"

  default_user_settings {
    execution_role = aws_iam_role.sagemaker.arn
  }
}
