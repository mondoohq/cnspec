# No failing fixture is possible

The check's `mql` is `terraform.resources("azurerm_monitor_diagnostic_setting").length > 0`,
which is exactly what the `filter` already requires (the filter only selects assets that
contain an `azurerm_monitor_diagnostic_setting` resource). Any asset that matches the filter
therefore has length > 0 and passes. There is no realistic Terraform config that matches the
filter yet fails the check.
