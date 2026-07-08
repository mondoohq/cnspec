resource openai 'Microsoft.CognitiveServices/accounts@2024-10-01' = {
  name: 'myopenaiaccount'
  location: 'eastus'
  kind: 'OpenAI'
  sku: {
    name: 'S0'
  }
  properties: {
    customSubDomainName: 'myopenaiaccount'
    publicNetworkAccess: 'Enabled'
    networkAcls: {
      defaultAction: 'Deny'
      ipRules: [
        {
          value: '203.0.113.0/24'
        }
      ]
    }
  }
}
