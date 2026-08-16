extension microsoftGraphV1

resource blockLegacyAuth 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Block legacy authentication'
  state: 'disabled'
  conditions: {
    clientAppTypes: [
      'exchangeActiveSync'
      'other'
    ]
    applications: {
      includeApplications: [
        'All'
      ]
    }
    users: {
      includeUsers: [
        'All'
      ]
    }
  }
  grantControls: {
    operator: 'OR'
    builtInControls: [
      'block'
    ]
  }
}
