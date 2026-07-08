This check is an existence check: the mql (`...where(type == "AWS::AccessAnalyzer::Analyzer").length > 0`)
is equivalent to its filter (`cloudformation.template.resources.contains(type == "AWS::AccessAnalyzer::Analyzer")`).
Any template that the filter selects (i.e. contains an Analyzer) makes length > 0 true, so no realistic
failing fixture exists. Only a pass fixture is provided.
