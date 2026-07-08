resource profile 'Microsoft.Cdn/profiles@2023-05-01' = {
  name: 'afd-prod-001'
  location: 'global'
  sku: {
    name: 'Premium_AzureFrontDoor'
  }
}

resource endpoint 'Microsoft.Cdn/profiles/afdEndpoints@2023-05-01' = {
  parent: profile
  name: 'ep-prod-001'
  location: 'global'
  properties: {
    enabledState: 'Enabled'
  }
}
