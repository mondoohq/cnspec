resource siteConfig 'Microsoft.Web/sites/config@2022-09-01' = {
  name: 'app-prod-001/web'
  properties: {
    cors: {
      allowedOrigins: [
        'https://portal.contoso.com'
        'https://app.contoso.com'
      ]
      supportCredentials: true
    }
  }
}
