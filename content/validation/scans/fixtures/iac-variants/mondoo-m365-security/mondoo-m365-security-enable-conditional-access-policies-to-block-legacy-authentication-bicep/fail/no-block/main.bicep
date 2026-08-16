extension microsoftGraphV1

resource legacyAuthPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: 'Require MFA for legacy clients'
  state: 'enabled'
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
      'mfa'
    ]
  }
}
