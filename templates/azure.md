# CLAUDE.md — Azure Extensions

Azure infrastructure, secrets, and CI/CD guidelines to add to your project's CLAUDE.md alongside the base template.
These can be added as sections in your CLAUDE.md or folded into Project Conventions.

## Secrets and Local Development

- Use `local.settings.json` for Azure Functions (not `.env`). Never commit it.
- `local.settings.json` can be encrypted — prefer encrypted when testing locally.
- Provide `local.settings.example.json` with dummy values as template.
- Be aware: AI tools with file access can read plaintext secret files. Prefer encrypted local config.
- For production: use Azure Key Vault with `@Microsoft.KeyVault(SecretUri=...)` resolution at startup.

## Managed Identity vs Connection Strings

- Detect explicitly: `USE_MANAGED_IDENTITY = !!STORAGE_ACCOUNT_NAME && !STORAGE_CONNECTION_STRING`.
- Never silently fall back between managed identity and connection string — fail fast if config is ambiguous.

```typescript
// Correct — explicit environment detection
if (USE_MANAGED_IDENTITY) {
    client = new TableClient(endpoint, tableName, credential);
} else {
    client = TableClient.fromConnectionString(connString, tableName);
}

// Incorrect — silent fallback masks configuration errors
const connString = process.env.AzureWebJobsStorage || 'UseDevelopmentStorage=true';
```

## Resource Naming (Azure CAF)

- Full words only, no abbreviations.
- Storage accounts: `st<shortname>${uniqueString}` (lowercase, no hyphens).
- Function apps: `<app>-${uniqueString(resourceGroup().id)}`.
- Key Vault: `kv-${uniqueString}` (not app-prefixed — security boundary).
- App Insights: `${functionAppName}-insights`.
- Log Analytics: `${functionAppName}-insights-ws`.

## Infrastructure as Code (Bicep)

- All resources defined in `infrastructure/main.bicep`.
- Sensitive parameters marked with `@secure()` decorator.
- Use `uniqueString(resourceGroup().id)` for globally unique names.
- Infrastructure README follows standard structure: Prerequisites, Quick Deploy, Parameters, Outputs, Monitoring, Cleanup.

## API Key Security

- Never log raw API keys — use truncated SHA-256 hash (8 hex chars) for identification.
- Three-tier model when needed: standard (operational fields only), restricted (PII opt-in), admin (full).

## Azure Blob Storage Gotchas

- `setMetadata()` **REPLACES** all metadata — spread operator has no effect. Explicitly set every key.
- Metadata keys are lowercased by Azure (`screenId` becomes `screenid`).
- All metadata values are strings — parse on read.
- Create helper functions for metadata operations to prevent these bugs.

```typescript
// WRONG — setMetadata() REPLACES all metadata, spread has no effect
await blockBlobClient.setMetadata({
    ...existingMetadata,
    screenid: screenId.toString()
});

// CORRECT — explicitly set all required keys
await blockBlobClient.setMetadata({
    displayorder: displayOrder.toString(),
    screenid: screenId.toString()
});
```

## Table Storage Patterns

- Use helper functions for client creation — never create clients inline.
- PartitionKey: logical domain grouping. RowKey: SHA-256 hash of unique identifier.

## CI/CD (GitHub Actions)

- Triggers: push to main + pull requests against main.
- Node.js: matrix test against versions 20 and 22.
- Go: multi-platform build matrix (darwin/linux/windows x amd64/arm64).
- Upload coverage to Codecov.
- Release workflow: tag-triggered, version injection, checksums, GitHub release with binaries.

## Deployment

- Function apps: `npm run deploy` or `func azure functionapp publish`.
- Multi-step deployments: deploy infrastructure first, then code, then event triggers (Event Grid needs function keys).
- Azure Function Apps support timer triggers natively. Static Web Apps (managed functions) do not — use Logic Apps with API key validation for scheduled tasks in SWA projects.
