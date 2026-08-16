resource "aws_appflow_flow" "salesforce_to_s3" {
  name    = "salesforce-to-s3"
  kms_arn = aws_kms_key.appflow.arn

  source_flow_config {
    connector_type = "Salesforce"

    source_connector_properties {
      salesforce {
        object = "Account"
      }
    }
  }

  destination_flow_config {
    connector_type = "S3"

    destination_connector_properties {
      s3 {
        bucket_name = aws_s3_bucket.appflow.bucket
      }
    }
  }

  task {
    source_fields     = ["Id"]
    task_type         = "Map"
    destination_field = "Id"

    connector_operator {
      salesforce = "NO_OP"
    }
  }

  trigger_config {
    trigger_type = "OnDemand"
  }
}
