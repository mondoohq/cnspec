# Compliant: model explainability job enables network isolation and inter-container encryption.
resource "aws_sagemaker_model_explainability_job_definition" "pass_example" {
  name     = "example-job"
  role_arn = "arn:aws:iam::123456789012:role/example"

  model_explainability_app_specification {
    image_uri  = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
    config_uri = "s3://example-bucket/analysis-config.json"
  }

  model_explainability_job_input {
    endpoint_input {
      endpoint_name = "example-endpoint"
      local_path    = "/opt/ml/processing/input"
    }
  }

  model_explainability_job_output_config {
    monitoring_outputs {
      s3_output {
        local_path = "/opt/ml/processing/output"
        s3_uri     = "s3://example-bucket/output"
      }
    }
  }

  job_resources {
    cluster_config {
      instance_count    = 1
      instance_type     = "ml.m5.large"
      volume_size_in_gb = 20
    }
  }

  network_config {
    enable_network_isolation                  = true
    enable_inter_container_traffic_encryption = true
  }
}
