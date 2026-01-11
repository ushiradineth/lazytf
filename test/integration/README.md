# Integration Tests

Run tagged integration tests with:

```bash
go test -tags=integration ./test/integration
```

These tests are expected to use local terraform fixtures and may require a working terraform binary.

## Fixtures

- `testdata/terraform/fixtures/basic`: single null resource for plan/apply workflows.
- `testdata/terraform/fixtures/module`: module-based resource to validate module addressing.

## Coverage Expectations

- Run init/plan/apply against real terraform using fixtures copied into temp directories.
- Use local TF data/cache directories so terraform writes stay within the test workspace.
- Validate standard text plan output and JSON streaming parsing where supported.
