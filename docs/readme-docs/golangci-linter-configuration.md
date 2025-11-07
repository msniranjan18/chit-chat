## .golangci.yml - Linter Configuration
### What it is:
Configuration file for golangci-lint, a fast Go linter that runs multiple linters in parallel.

### Why it's needed:
Code Quality: Enforces consistent coding standards

Error Prevention: Catches common bugs before runtime

Team Consistency: Everyone follows the same rules

Automated Reviews: Can be integrated into CI/CD

### What it does:
```yaml
# .golangci.yml example
linters:
  enable:
    - errcheck         # Checks for unhandled errors
    - govet            # Official Go vet tool
    - staticcheck      # Advanced static analysis
    - gosimple         # Suggests simplifications
    - ineffassign      # Detects ineffective assignments
    - unused           # Finds unused code
    - gofmt           # Checks code formatting
    - goimports       # Checks import formatting

issues:
  exclude-use-default: false
  max-same-issues: 0  # Show all issues
```

### Linters it runs:
Linter	    |      Purpose
----------------------------------------------------------------
errcheck    |      Finds unhandled errors: json.Marshal(data) ❌
govet		|      Official Go analysis tool
staticcheck |      Advanced bug detection
gosimple    |      Suggests simpler code
ineffassign |      x = 5 (but x not used)
unused		|      Unused variables/functions
gofmt		|      Code formatting
goimports   |      Import organization


### Example issues it catches:
Unhandled errors (errcheck):

```go
// BAD - error not handled
json.Marshal(data)

// GOOD
if err := json.Marshal(data); err != nil {
    return err
}
```

### Formatting issues (gofmt):

```go
// BAD - inconsistent spacing
if err!=nil{

// GOOD
if err != nil {
```

### Unused code (unused):

```go
// BAD - unused variable
var unusedVar = "hello"

// GOOD - variable is used
var usedVar = "hello"
fmt.Println(usedVar)
```

### When to use it:
- Pre-commit hooks: Run before committing code
- CI/CD pipelines: Fail builds on linting errors
- Editor integration: Real-time feedback in VS Code/Vim
- Code reviews: Automated quality checks

### Usage:
```bash
# Run all linters
make lint

# Or directly
golangci-lint run ./...

# Run with auto-fix
golangci-lint run --fix ./...
```