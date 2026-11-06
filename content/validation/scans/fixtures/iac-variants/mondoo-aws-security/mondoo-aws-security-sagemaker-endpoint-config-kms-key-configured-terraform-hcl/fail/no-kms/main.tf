# The hosting instance volume has no KMS key.
resource "aws_sagemaker_endpoint_configuration" "example" {
  name = "example-config"

  production_variants {
    variant_name           = "AllTraffic"
    model_name             = "example-model"
    initial_instance_count = 1
    instance_type          = "ml.m5.large"
  }
}
