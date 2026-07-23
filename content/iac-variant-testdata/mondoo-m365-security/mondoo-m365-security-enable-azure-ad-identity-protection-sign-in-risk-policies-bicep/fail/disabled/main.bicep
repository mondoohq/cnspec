extension microsoftGraphV1

@description('Display name for the conditional access policy')
param policyName string = 'Require MFA for risky sign-ins'

// The policy targets sign-in risk and requires MFA, but it is only in
// report-only/disabled state, so it is not enforced.
resource signInRiskPolicy 'Microsoft.Graph/conditionalAccessPolicies@v1.0' = {
  displayName: policyName
  state: 'disabled'
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
