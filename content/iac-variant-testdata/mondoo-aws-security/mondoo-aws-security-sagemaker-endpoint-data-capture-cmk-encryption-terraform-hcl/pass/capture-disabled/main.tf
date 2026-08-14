# Capture turned off, so no inference payloads are written and there is nothing
# to encrypt.
resource "aws_sagemaker_endpoint_configuration" "scoring" {
  name = "scoring-config"

  production_variants {
    variant_name           = "primary"
    model_name             = aws_sagemaker_model.scoring.name
    initial_instance_count = 1
    instance_type          = "ml.m5.large"
  }

  data_capture_config {
    enable_capture              = false
    initial_sampling_percentage = 0
    destination_s3_uri          = "s3://example-model-capture/scoring/"

    capture_options {
      capture_mode = "Input"
    }
  }
}
