No failing input is expressible on terraform-hcl: container_definitions must be a
jsonencode([...]) JSON string, and the terraform-hcl provider (both v13.27.1 and the
harness's v13.27.4) evaluates a top-level jsonencode LIST to [] (empty), making the
.all() over container definitions vacuously true. This is a provider bug (jsonencode of a
list literal), tracked by the provider PR. jsonencode({map}) resolves correctly; only the
list form is broken.
