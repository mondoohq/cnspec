This check's mql is `cloudformation.template.resources.where(type == "AWS::SecretsManager::RotationSchedule").length > 0`,
which is an existence check equivalent to the filter itself
(`cloudformation.template.resources.contains(type == "AWS::SecretsManager::RotationSchedule")`).
Any template that matches the filter (contains a RotationSchedule) necessarily satisfies the mql,
so no realistic failing fixture exists. This is a genuine pass-only existence check.
