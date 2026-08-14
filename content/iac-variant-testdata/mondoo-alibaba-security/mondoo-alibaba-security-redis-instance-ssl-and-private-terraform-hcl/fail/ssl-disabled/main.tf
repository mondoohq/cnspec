# Redis traffic, including the AUTH password, crosses the network in cleartext.
resource "alicloud_kvstore_instance" "cache" {
  db_instance_name = "prod-redis"
  instance_class   = "redis.master.small.default"
  instance_type    = "Redis"
  engine_version   = "7.0"
  vswitch_id       = alicloud_vswitch.cache.id
  ssl_enable       = "Disable"
}
