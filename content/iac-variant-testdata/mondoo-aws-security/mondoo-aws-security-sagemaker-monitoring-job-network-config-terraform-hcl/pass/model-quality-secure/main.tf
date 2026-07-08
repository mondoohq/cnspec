# Compliant: model quality job enables network isolation and inter-container encryption.
resource "aws_sagemaker_model_quality_job_definition" "pass_example" {
  name = "example-job"

  model_quality_app_specification {
    image_uri          = "123456789012.dkr.ecr.us-east-1.amazonaws.com/example:latest"
    problem_type       = "BinaryClassification"
  }

  model_quality_job_input {
    endpoint_input {
      endpoint_name = "example-endpoint"
      local_path    = "/opt/ml/processing/input"
    }
    ground_truth_s3_input {
      s3_uri = "s3://example-bucket/ground-truth"
    }
  }

  model_quality_job_output_config {
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

  role_arn = "arn:aws:iam::123456789012:role/example"

  network_config {
    enable_network_isolation                  = true
    enable_inter_container_traffic_encryption = true
  }
}
