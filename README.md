# Domainry Notification SDK

`domainry-notification-sdk` is the deployment-neutral contract between a
Domainry Runtime and Notification running either as an embedded Go module or a
remote SaaS service.

The SDK owns stable request, response, pagination, error, Factory and Binding
contracts. It owns no Notification implementation, SQL persistence, worker
runtime, Identity implementation, Runtime model, or Connector provider.

## Deployment invariants

- Module and Remote Factories expose the same `Binding`.
- Module mode borrows the project's database pool and never closes it.
- SaaS mode uses a separately authenticated remote service.
- User use cases carry the original Identity access token. The selected
  Notification implementation verifies it through its Identity SDK Binding.
- Publication authority comes from the application-scoped Binding and every
  intent carries an explicit workspace and deterministic source identity.

Run `go test ./...` before publishing an immutable module version.
