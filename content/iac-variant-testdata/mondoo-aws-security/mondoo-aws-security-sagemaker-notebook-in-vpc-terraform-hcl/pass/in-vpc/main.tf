resource "aws_sagemaker_notebook_instance" "research" {
  name                   = "research-notebook"
  role_arn               = aws_iam_role.sagemaker.arn
  instance_type          = "ml.t3.medium"
  subnet_id              = aws_subnet.private_a.id
  security_groups        = [aws_security_group.sagemaker.id]
  direct_internet_access = "Disabled"
  kms_key_id             = aws_kms_key.sagemaker.arn
}
