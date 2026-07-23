No failing input exists for this variant. Its MQL is
`terraform.resources("aws_config_configuration_aggregator").length > 0`, which is
logically identical to its filter (the resource must be present for the check to
run). Any fixture that matches the filter therefore passes; omitting the resource
skips the check rather than failing it.
