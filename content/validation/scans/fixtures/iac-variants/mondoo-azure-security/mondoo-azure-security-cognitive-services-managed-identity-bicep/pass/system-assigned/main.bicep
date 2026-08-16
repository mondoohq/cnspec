resource openai 'Microsoft.CognitiveServices/accounts@2024-10-01' = {
  name: 'myopenaiaccount'
  location: 'eastus'
  kind: 'OpenAI'
  sku: {
    name: 'S0'
  }
  identity: {
    type: 'SystemAssigned'
  }
  properties: {
    customSubDomainName: 'myopenaiaccount'
  }
}
