# Reproduction

Run:

```bash
go test ./internal/transport -count=1 -run '^TestBug02NilServiceRejectsIngestion$'
```

Expected: an event request cannot be accepted when the HTTP server has no application service.

Actual: the test fails because the request returns HTTP 202.
