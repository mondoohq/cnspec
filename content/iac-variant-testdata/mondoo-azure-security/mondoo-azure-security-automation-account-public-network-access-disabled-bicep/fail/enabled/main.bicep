resource automationAccount 'Microsoft.Automation/automationAccounts@2023-11-01' = {
  name: 'example-automation'
  location: 'eastus'
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    sku: {
      name: 'Basic'
    }
    publicNetworkAccess: true
    disableLocalAuth: true
  }
}
