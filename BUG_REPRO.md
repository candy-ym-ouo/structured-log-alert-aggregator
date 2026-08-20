# Reproduction

Run:

```bash
go test ./internal/store -count=1 -run '^TestBug01IngestDoesNotChangeStoredPolicyChannel$'
```

Expected: processing an event leaves the stored policy notification channel set to `email`.

Actual: the test fails because the stored channel becomes `audit`.
