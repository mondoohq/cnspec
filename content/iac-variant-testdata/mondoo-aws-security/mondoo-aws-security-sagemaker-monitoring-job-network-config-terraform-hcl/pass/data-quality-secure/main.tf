# Compliant: data quality job enables network isolation and inter-container encryption.
resource "aws_sagemaker_data_quality_job_definition" "pass_example" {
  name = "example-job"

  data_quality_app_specification {
    image_uri = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
  }

  network_config {
    enable_network_isolation                 = true
    enable_inter_container_traffic_encryption = true
  }
}
