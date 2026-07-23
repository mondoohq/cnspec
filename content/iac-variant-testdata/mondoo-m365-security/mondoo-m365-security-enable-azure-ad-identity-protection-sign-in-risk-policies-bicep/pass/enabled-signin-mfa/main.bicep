extension microsoftGraphV1

@description('Display name for the conditional access policy')
param policyName string = 'Require MFA for risky sign-ins'

resource signInRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: policyName
  state: 'enabled'
  conditions: {
    clientAppTypes: [
      'all'
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
    signInRiskLevels: [
      'high'
      'medium'
    ]
  }
  grantControls: {
    operator: 'OR'
    builtInControls: [
      'mfa'
    ]
  }
}
