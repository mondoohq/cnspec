# Compliant: notebook instance requires IMDSv2.
resource "aws_sagemaker_notebook_instance" "pass_example" {
  name          = "example-notebook"
  instance_type = "ml.t3.medium"
  role_arn      = "arn:aws:iam::123456789012:role/example"

  instance_metadata_service_configuration {
    minimum_instance_metadata_service_version = "2"
  }
}
