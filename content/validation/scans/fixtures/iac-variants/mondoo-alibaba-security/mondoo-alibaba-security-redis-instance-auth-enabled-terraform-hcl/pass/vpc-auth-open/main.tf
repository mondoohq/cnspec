# vpc_auth_mode "Open" means password authentication is required inside the VPC.
resource "alicloud_kvstore_instance" "cache" {
  db_instance_name = "prod-redis"
  instance_class   = "redis.master.small.default"
  instance_type    = "Redis"
  engine_version   = "7.0"
  vswitch_id       = alicloud_vswitch.cache.id
  vpc_auth_mode    = "Open"
  password         = var.redis_password
}
