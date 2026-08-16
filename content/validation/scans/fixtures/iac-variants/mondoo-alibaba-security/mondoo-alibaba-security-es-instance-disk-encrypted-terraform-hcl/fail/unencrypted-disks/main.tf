# Disk encryption is chosen at creation and cannot be turned on later, so this
# cluster stores its indexes in the clear for its whole life.
resource "alicloud_elasticsearch_instance" "logs" {
  instance_charge_type     = "PostPaid"
  data_node_amount         = 2
  data_node_spec           = "elasticsearch.sn2ne.large"
  data_node_disk_size      = 20
  data_node_disk_type      = "cloud_ssd"
  data_node_disk_encrypted = false
  version                  = "7.10_with_X-Pack"
  vswitch_id               = "vsw-example"
  password                 = var.elasticsearch_password
}
