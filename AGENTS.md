# Agent Guidelines

This project uses Go modules. When making changes:

- If you modify any Go files, format the changed code with `gofmt -w`.
- Run `go test ./...` only when Go code is updated and ensure it succeeds. Include the test output in the PR description.
- Keep commit messages concise.
- PR description should have a summary of changes and testing details.
## Testing Conventions
- Tests should be table-driven. Declare a slice of test cases with a `name` field and execute each case using `t.Run`.
- Compare returned values and errors with `reflect.DeepEqual` as done in `globalFunctions_test.go`.
- When a function returns both a result and an error, store expected values in `want` and `wantErr` fields, mirroring `TestPathFuncAndIsPathFunc`.
- Use custom error types instead of `fmt.Errorf` when equality checks are required.
- Place tests in the `*_test.go` file that corresponds to the source file and keep them in the same order as the functions they verify.
