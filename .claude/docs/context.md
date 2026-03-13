# `context.Context` Usage

Module-wide rules for Go `context.Context` + package-specific payload objects.

## Core Split

- Use stdlib `context.Context` for cancellation, deadlines, tracing, request-scoped values.
- Use descriptive package payload structs for package configuration/state only.
- Default to unexported payload type + helper names.
- NEVER name package payload type `Context`.
- NEVER store `context.Context` inside package payload struct.
- NEVER embed `context.Context` in package payload struct.
- NEVER make package payload implement `context.Context`.
- Pass stdlib `context.Context` as explicit parameter through call chain.

## Exposure Rules

- Keep payload type, option type, constructor, accessor, helper pair unexported by default.
- Prefer exported direct mutators on `context.Context` over exporting payload type or option type.
- If callers only need to set package-scoped values, export helpers like `WithX(ctx, value) context.Context`.
- Export initializer only when external callers must attach package payload before calling API that accepts `context.Context`.
- Export accessor only when external callers or sibling packages must inspect/merge payload already attached to `context.Context`.
- Export payload type only when exported accessor returns it or callers must mutate/read it directly.
- If helper is internal execution plumbing, keep it unexported even when read side exists elsewhere.
- If any part must be exported, use descriptive names. NEVER export bare `Context`, `NewContext`, `GetContext`, `ContextOption`.

## Carrier Pattern

- Prefer initializer-style carrier helpers, e.g. `WithXPathContext(parent, opts ...XPathContextOption) context.Context`.
- Prefer simpler direct mutators when possible, e.g. `WithXPathNamespaces(parent, ns) context.Context`.
- Use descriptive accessor helpers, e.g. `GetXPathContext(ctx context.Context) *XPathContext`.
- Accept parent stdlib context as first parameter.
- Return derived context from `context.WithValue(parent, contextKey{}, payload)`.
- Keep payload type descriptive: `type XPathContext struct { ... }`.
- Keep carrier key package-private: `type contextKey struct{}`.
- Use one unique key type per stored payload kind.
- Return `nil` from getter when payload absent or wrong type.
- NEVER panic on missing payload.

## Payload Rules

- Store only package data in payload: namespaces, variables, options, evaluators, limits, resolvers, etc.
- Keep payload independent from cancellation/deadline lifecycle.
- Copy caller-owned mutable maps/slices before storing when later mutation would change behavior unexpectedly.
- Precompute reusable derived state inside payload when safe.
- Do not thread stdlib context through payload methods or fields.

## Call-Site Rules

- Exported operations accept `ctx context.Context`.
- Extract package payload via descriptive accessor, e.g. `GetXPathContext(ctx)`.
- Continue passing same stdlib `ctx` downward for cancellation + other package values.
- Wrap existing `ctx`; NEVER replace with `context.Background()` or `context.TODO()` in library path.
- Add package payload only at boundary where caller config enters package.
- Allow multiple packages to attach independent payloads to same stdlib context.
- Prefer chaining direct mutators over constructing/exporting mutable payload objects.

## Internal Extra State

- For internal-only execution state, use separate unexported key type + helper pair.
- Pattern: `withXContext(ctx, state) context.Context` + `getXContext(ctx) *stateType`.
- Keep helper unexported unless callers must read/write that state.
- NEVER overload exported package payload getter with transient internal state.

## Current Repo Pattern

- `xpath1` now uses direct mutators such as `WithNamespaces(ctx, ns)` and `WithAdditionalVariables(ctx, vars)`.
- `xpath1` keeps `FunctionContext` + `GetFunctionContext(ctx)` public because custom function implementations need read-only evaluation state.
- `xpath1` hides its eval-config carrier and getter.
- `xpath3` now also uses direct mutators such as `WithNamespaces(ctx, ns)` and `WithDefaultLanguage(ctx, lang)`.
- `xpath3` hides its eval-config carrier and getter.
- `xpath3` internal function-evaluation state already follows unexported helper pattern: `withFnContext` + `getFnContext`.
- Future refactors should prefer `WithX(ctx, value)` style mutators when callers do not need raw payload access.
- Treat public carrier/accessor APIs as exception driven by caller need, not default style.

## Naming

- Unexported default: `xpathContext`, `withXPathContext`, `getXPathContext`, `xpathContextOption`.
- Exported exception: `XPathContext`, `WithXPathContext`, `GetXPathContext`, `XPathContextOption`.
- Preferred exported mutator style: `WithXPathNamespaces`, `WithXPathVariables`, `WithXPathFunction`.
- Mutators: `WithX`.
- Internal helper keys: `<name>ContextKey`.

## Reference Shape

```go
type contextKey struct{}

type XPathContext struct {
    // package config only
}

type XPathContextOption func(*XPathContext)

func WithXPathContext(parent context.Context, opts ...XPathContextOption) context.Context {
    c := &XPathContext{}
    for _, opt := range opts {
        opt(c)
    }
    return context.WithValue(parent, contextKey{}, c)
}

func GetXPathContext(ctx context.Context) *XPathContext {
    c, _ := ctx.Value(contextKey{}).(*XPathContext)
    return c
}
```

## Anti-Patterns

- `type Context struct { ... }`
- Exporting payload type when direct `WithX(ctx, value)` helpers would suffice
- `type XPathContext struct { parent context.Context }`
- `type XPathContext struct { context.Context }`
- `func NewContext(...) *Context`
- `func NewXPathContext(parent context.Context, ...) context.Context`
- `func (c *XPathContext) Deadline() ...`
- Storing cancellation functions in package payload
- Reusing one key type for unrelated payload types
