# Non-compliant: account aggregation source (all_regions=false) via dynamic block.
variable "accounts" {
  type    = list(string)
  default = ["123456789012"]
}

resource "aws_config_configuration_aggregator" "fail_dynamic" {
  name = "fail-dynamic"

  dynamic "account_aggregation_source" {
    for_each = var.accounts
    content {
      account_ids = [account_aggregation_source.value]
      regions     = ["us-east-1"]
      all_regions = false
    }
  }
}
