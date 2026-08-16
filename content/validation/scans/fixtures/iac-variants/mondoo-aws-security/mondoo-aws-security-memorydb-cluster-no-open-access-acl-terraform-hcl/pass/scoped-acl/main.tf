resource "aws_memorydb_cluster" "sessions" {
  name                     = "sessions"
  acl_name                 = aws_memorydb_acl.app.name
  node_type                = "db.t4g.small"
  num_shards               = 2
  security_group_ids       = [aws_security_group.memorydb.id]
  subnet_group_name        = aws_memorydb_subnet_group.private.name
  tls_enabled              = true
  kms_key_arn              = aws_kms_key.memorydb.arn
}
