This check is a pure existence check: mql is
`cloudformation.template.resources.where(type == "AWS::Config::ConfigurationAggregator").length > 0`
which is exactly what the filter already requires. Any template that satisfies the filter
(contains an AWS::Config::ConfigurationAggregator) makes the mql true. There is no realistic
fixture that matches the filter yet fails the check. pass_only.
