# The hosting instance volume is encrypted with a named KMS key.
resource "aws_sagemaker_endpoint_configuration" "example" {
  name        = "example-config"
  kms_key_arn = "arn:aws:kms:us-east-1:111122223333:key/1234abcd-12ab-34cd-56ef-1234567890ab"

  production_variants {
    variant_name           = "AllTraffic"
    model_name             = "example-model"
    initial_instance_count = 1
    instance_type          = "ml.m5.large"
  }
}
