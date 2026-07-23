# Non-compliant: data quality job network_config disables encryption.
resource "aws_sagemaker_data_quality_job_definition" "fail_example" {
  name = "example-job"

  data_quality_app_specification {
    image_uri = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
  }

  network_config {
    enable_network_isolation                 = false
    enable_inter_container_traffic_encryption = false
  }
}
