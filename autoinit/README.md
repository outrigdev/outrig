# Outrig Auto-Init

This package provides automatic initialization of Outrig when imported, similar to how `net/http/pprof` works.

## Usage

Simply import this package with a blank identifier to automatically initialize Outrig with default settings:

```go
import _ "github.com/outrigdev/outrig/autoinit"
```

This is equivalent to calling:

```go
outrig.Init("", nil)
```

## When to Use

Use this package when you want the simplest possible Outrig integration with zero configuration. The auto-init approach is perfect for:

- Quick debugging sessions
- Development environments
- Applications where you don't need custom Outrig configuration

## When NOT to Use

Don't use this package if you need:

- Custom application names
- Custom configuration settings
- Error handling from the Init call
- Control over when Outrig is initialized

In those cases, use the regular `outrig.Init()` function directly.