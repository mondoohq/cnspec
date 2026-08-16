resource automationAccount 'Microsoft.Automation/automationAccounts@2023-11-01' = {
  name: 'myautomationaccount'
  location: 'eastus'
  properties: {
    sku: {
      name: 'Basic'
    }
  }
}

resource environmentVar 'Microsoft.Automation/automationAccounts/variables@2023-11-01' = {
  parent: automationAccount
  name: 'environment'
  properties: {
    value: '"production"'
  }
}
