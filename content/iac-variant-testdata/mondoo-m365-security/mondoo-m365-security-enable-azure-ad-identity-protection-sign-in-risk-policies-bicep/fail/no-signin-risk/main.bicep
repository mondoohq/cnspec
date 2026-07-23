extension microsoftGraphV1

@description('Display name for the conditional access policy')
param policyName string = 'Require MFA for all users'

// This enabled policy requires MFA but does not scope on sign-in risk levels,
// so it does not satisfy the Identity Protection sign-in risk requirement.
resource requireMfaPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
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
  }
  grantControls: {
    operator: 'OR'
    builtInControls: [
      'mfa'
    ]
  }
}
