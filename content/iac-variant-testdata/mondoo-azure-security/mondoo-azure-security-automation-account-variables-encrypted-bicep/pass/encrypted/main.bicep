resource automationAccount 'Microsoft.Automation/automationAccounts@2023-11-01' = {
  name: 'myautomationaccount'
  location: 'eastus'
  properties: {
    sku: {
      name: 'Basic'
    }
  }
}

resource connectionStringVar 'Microsoft.Automation/automationAccounts/variables@2023-11-01' = {
  parent: automationAccount
  name: 'sqlConnectionString'
  properties: {
    value: '"Server=tcp:example.database.windows.net;"'
    isEncrypted: true
  }
}
