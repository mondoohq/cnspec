# The prod VPC has a flow log, but the staging VPC declared alongside it has
# none, so staging traffic leaves no record.
resource "alicloud_vpc" "prod" {
  vpc_name   = "prod"
  cidr_block = "10.0.0.0/16"
}

resource "alicloud_vpc" "staging" {
  vpc_name   = "staging"
  cidr_block = "10.1.0.0/16"
}

resource "alicloud_vpc_flow_log" "prod" {
  flow_log_name  = "prod-vpc-flow"
  resource_id    = alicloud_vpc.prod.id
  resource_type  = "VPC"
  traffic_type   = "All"
  project_name   = "vpc-flow-logs"
  log_store_name = "prod-vpc"
  status         = "Active"
}
