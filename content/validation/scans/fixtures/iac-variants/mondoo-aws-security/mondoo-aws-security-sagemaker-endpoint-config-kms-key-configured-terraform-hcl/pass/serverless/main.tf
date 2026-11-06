# A serverless endpoint has no attached storage volume to encrypt.
resource "aws_sagemaker_endpoint_configuration" "example" {
  name = "example-config"

  production_variants {
    variant_name = "AllTraffic"
    model_name   = "example-model"

    serverless_config {
      max_concurrency   = 5
      memory_size_in_mb = 2048
    }
  }
}
