resource siteConfig 'Microsoft.Web/sites/config@2022-09-01' = {
  name: 'app-prod-001/web'
  properties: {
    minTlsCipherSuite: 'TLS_AES_128_GCM_SHA256'
  }
}
