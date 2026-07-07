## Purpose: TBD

## Requirements

### Requirement: Each github kind owns its full gh invocation
Each github kind file SHALL declare its complete `gh` CLI invocation: the argument slice passed to `exec.Command("gh", ...)`, its own payload struct for decoding the JSON response, and the mapping from decoded payload fields to `workitem.WorkItem` fields (including any kind-specific fields such as `TargetBranch`). A kind file SHALL NOT rely on a shared helper to choose its args, its `--json` field list, or its WorkItem field mapping.

#### Scenario: Issue kind declares its own args and payload
- **WHEN** the `issue` provider's `Fetch` is called with a valid `SourceRef`
- **THEN** it executes `gh issue view <ref> --json title,body,url` using an argument slice declared in `issue.go`, decodes the output into a payload struct declared in `issue.go`, and builds a `WorkItem` with `Title`, `Description`, and `URL` populated from that struct

#### Scenario: PR kind owns its TargetBranch mapping
- **WHEN** the `pr` provider's `Fetch` is called with a valid `SourceRef`
- **THEN** it includes `headRefName` in its own `--json` field list declared in `pr.go`, decodes into its own payload struct declared in `pr.go`, and sets `WorkItem.TargetBranch` from that field directly in `pr.go`

### Requirement: Shared view-shaped helpers are opt-in
`common.go` MAY expose helpers (`runView`, `baseWorkItem`) that encapsulate the exec + decode + normalize + base-build transform shared by kinds whose `gh` command fits the `gh <kind> view <ref> --json <fields>` shape. These helpers SHALL be opt-in: a kind whose `gh` invocation does not fit the view shape (e.g. `gh api graphql`, `gh release view`) SHALL bypass them and call `exec.Command` directly with its own payload struct, touching no shared code.

#### Scenario: View-shaped kind reuses runView and baseWorkItem
- **WHEN** a kind whose `gh` command is `gh <kind> view <ref> --json <fields>` calls `runView` with its declared args
- **THEN** it receives a decoded `viewPayload` and may call `baseWorkItem` to obtain a `WorkItem` with `Source`, `Identifier`, `Title`, `Description`, and `URL` populated, adding any kind-specific fields itself

#### Scenario: Non-view-shaped kind bypasses shared helpers
- **WHEN** a kind uses a `gh` invocation that is not the view shape (e.g. `gh api graphql ...` or `gh release view <tag> --json tagName,body`)
- **THEN** it calls `exec.Command` directly in its own kind file, declares its own payload struct, and performs its own decode and WorkItem mapping without invoking `runView` or `baseWorkItem`

### Requirement: No kind-aware branches in shared code
`common.go` SHALL NOT contain any conditional branch (if/switch) keyed on the kind string, the `commandName`, or any per-kind field list. All kind-specific logic (which `--json` fields to request, which WorkItem fields to populate) SHALL live in the kind file that owns it.

#### Scenario: common.go is kind-agnostic
- **WHEN** `common.go` is inspected
- **THEN** it contains no `if` or `switch` referencing `kind`, `commandName`, or a per-kind field list; the only kind-specific string (`sourceRef.Kind`) appears in shared error-message formatting, never in a control-flow branch
