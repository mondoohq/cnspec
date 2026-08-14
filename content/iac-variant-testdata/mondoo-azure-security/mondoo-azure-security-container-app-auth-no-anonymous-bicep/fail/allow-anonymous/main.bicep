// The auth config exists but serves anonymous requests, so the platform
// performs no access control.
resource authConfig 'Microsoft.App/containerApps/authConfigs@2024-03-01' = {
  name: 'current'
  properties: {
    platform: {
      enabled: true
    }
    globalValidation: {
      unauthenticatedClientAction: 'AllowAnonymous'
    }
  }
}
