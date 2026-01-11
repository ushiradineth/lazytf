# E2E Tests

Run tagged end-to-end tests with:

```bash
go test -tags=e2e ./test/e2e
```

These tests are expected to use real terraform projects and may be slower to execute.

Run the e2e suite with the race detector (slower):

```bash
go test -race -tags=e2e ./test/e2e
```
