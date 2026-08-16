# Without kms_arn the flow falls back to the AWS-managed AppFlow key, so the
# data in transit through AppFlow is not under customer key control.
resource "aws_appflow_flow" "salesforce_to_s3" {
  name = "salesforce-to-s3"

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

  trigger_config {
    trigger_type = "OnDemand"
  }
}
