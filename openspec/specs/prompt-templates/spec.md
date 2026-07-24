# prompt-templates

## Purpose: TBD

## Requirements

### Requirement: Prompts are built from template files

The prompt builder SHALL render the final worker prompt from template files using Go's `text/template` package. No prompt text SHALL be hard-coded in Go source beyond the embedded template files. The builder SHALL produce the prompt by resolving a template from the override hierarchy, executing it with template data, and returning the result.

#### Scenario: Default implementation prompt for a GitHub issue

- **WHEN** `Build` is called with a `WorkItem` whose `Source.System` is `github`, `Source.Kind` is `issue`, `Mode` is `ModeImplement`, and no override text and no `--pr` flag
- **THEN** the returned prompt starts with `Implement GitHub issue #<ref>.` followed by the work item context, with no PR creation instruction

#### Scenario: Default review prompt for a GitHub PR

- **WHEN** `Build` is called with a `WorkItem` whose `Source.System` is `github`, `Source.Kind` is `pr`, `Mode` is `ModeReview`, and no override text
- **THEN** the returned prompt starts with `Review GitHub PR #<ref>.` followed by the work item context and the review-only instruction

#### Scenario: Custom prompt source produces no source label

- **WHEN** `Build` is called with a `WorkItem` whose `Source.System` is `prompt` and `Source.Kind` is `prompt`, with no override text
- **THEN** the returned prompt contains the work item context without a leading `Implement` or `Review` label line

### Requirement: Three template types cover implement, review, and PR instructions

The system SHALL ship three template types: `implement` (for the implementation frame), `review` (for the review frame), and `pr` (for the PR creation instruction). Each type SHALL have a generic default template embedded in the binary.

#### Scenario: Implementation task uses implement template

- **WHEN** a work item with `ModeImplement` is built
- **THEN** the `implement` template type is resolved and rendered

#### Scenario: Review task uses review template

- **WHEN** a work item with `ModeReview` is built
- **THEN** the `review` template type is resolved and rendered

#### Scenario: PR instruction uses pr template

- **WHEN** `--pr` is set on an implementation task
- **THEN** the `pr` template type is resolved and rendered, and its output is passed to the implement template as the `PRInstruction` data field

### Requirement: Host-specific template overrides exist for GitHub, GitLab, and Bitbucket

For each template type, the system SHALL ship host-specific override templates for `github`, `gitlab`, and `bitbucket`. Host-specific templates are named `{type}-{host}.tmpl`. Generic templates are named `{type}.tmpl`. The host-specific PR templates SHALL name the concrete CLI tool for that host (`gh` for GitHub, `glab` for GitLab). The host-specific implement and review templates SHALL be structurally identical to their generic counterparts by default, present for per-host customization.

#### Scenario: GitHub host-specific PR template uses gh

- **WHEN** the resolved repository host is `github` and the `pr` template is rendered
- **THEN** the output references `gh` as the CLI tool for creating the PR

#### Scenario: GitLab host-specific PR template uses glab

- **WHEN** the resolved repository host is `gitlab` and the `pr` template is rendered
- **THEN** the output references `glab` as the CLI tool for creating the merge request

#### Scenario: Unknown host falls back to generic PR template

- **WHEN** the resolved repository host is `unknown` and the `pr` template is rendered
- **THEN** the output instructs the agent to detect the git remote host and use the corresponding CLI tool

#### Scenario: Host-specific implement template matches generic by default

- **WHEN** the embedded `implement-github.tmpl` is rendered with the same data as the embedded `implement.tmpl`
- **THEN** the output is identical

### Requirement: Template resolution follows a layered override hierarchy

Template resolution SHALL check paths in this order, using the first file that exists:

1. `./wtmag-prompts/{type}-{host}.tmpl` (project, host-specific)
2. `~/.config/wtmag/prompts/{type}-{host}.tmpl` (global, host-specific)
3. `./wtmag-prompts/{type}.tmpl` (project, generic)
4. `~/.config/wtmag/prompts/{type}.tmpl` (global, generic)
5. Embedded default `{type}.tmpl`

If no disk-based override is found, the embedded default for the type SHALL be used. Project paths SHALL be resolved relative to the current working directory's project root (same root used by the config system).

#### Scenario: Project host-specific override takes precedence

- **WHEN** `./wtmag-prompts/pr-github.tmpl` exists AND `~/.config/wtmag/prompts/pr-github.tmpl` exists AND the host is `github`
- **THEN** the project file is used

#### Scenario: Global host-specific overrides project generic

- **WHEN** `~/.config/wtmag/prompts/pr-github.tmpl` exists AND `./wtmag-prompts/pr.tmpl` exists AND the host is `github`
- **THEN** the global host-specific file is used (host-specific beats generic across layers)

#### Scenario: No disk overrides, embedded default is used

- **WHEN** no override files exist on disk for the given type and host
- **THEN** the embedded default template for the type is used

#### Scenario: Project generic overrides global generic

- **WHEN** `./wtmag-prompts/implement.tmpl` exists AND `~/.config/wtmag/prompts/implement.tmpl` exists AND no host-specific files exist
- **THEN** the project generic file is used

### Requirement: The --pr flag controls PR instruction injection

The `create` command SHALL accept a `--pr` boolean flag. When `--pr` is set on an implementation task, the PR template SHALL be rendered and its output passed to the implement template as the `PRInstruction` field. When `--pr` is not set on an implementation task, no PR instruction SHALL be injected and the `PRInstruction` field SHALL be empty. Passing `--pr` on a review task SHALL produce an error: `--pr is only valid with implementation tasks; review tasks already inspect a PR`.

#### Scenario: Implementation with --pr injects PR instruction

- **WHEN** `Build` is called with `ModeImplement`, `--pr` set to true, and the repository host is `github`
- **THEN** the output includes a PR creation instruction that references `gh`

#### Scenario: Implementation without --pr does not inject PR instruction

- **WHEN** `Build` is called with `ModeImplement` and `--pr` set to false
- **THEN** the output does not contain any PR creation or delivery instruction

#### Scenario: --pr on review task errors

- **WHEN** `Build` is called with `ModeReview` and `--pr` set to true
- **THEN** it returns an error whose message contains `--pr is only valid with implementation tasks`

### Requirement: Override text replaces the frame content

When the `--prompt` override text is provided, the frame template SHALL use the override text in place of the default frame content (the `Implement {label}.\n\n{context}` or `Review {label}.\n\n{context}` prefix). The work item context and PR instruction (if `--pr`) SHALL still be appended. For `prompt:prompt` source work items, the override text replaces the entire frame content.

#### Scenario: Override text on GitHub issue implementation

- **WHEN** `Build` is called with `ModeImplement`, override text `"Focus on tests"`, and `--pr` set to true
- **THEN** the output starts with the override text, followed by the work item context (if any), followed by the PR instruction

#### Scenario: Override text on prompt source

- **WHEN** `Build` is called with a `prompt:prompt` work item, override text `"Custom instructions"`, and `--pr` set to false
- **THEN** the output is the override text with no source label and no PR instruction

### Requirement: Reference backlink is resolved in Go and passed as template data

The reference backlink note (e.g., `Refs #456` for GitHub issues on GitHub host) SHALL be computed in Go from the work item source and repository host, and passed to the PR template as the `Reference` data field. The PR template SHALL NOT contain logic to compute the reference. The reference SHALL be empty when the source is not a GitHub issue or the host is not GitHub.

#### Scenario: GitHub issue on GitHub host gets reference backlink

- **WHEN** the PR template is rendered for a `github:issue` work item on a `github` host
- **THEN** the `Reference` field contains `, make sure the PR body references GitHub issue #<ref> with a non-closing reference like 'Refs #<ref>'`

#### Scenario: GitHub PR on GitHub host gets no reference backlink

- **WHEN** the PR template is rendered for a `github:pr` work item on a `github` host
- **THEN** the `Reference` field is empty

#### Scenario: Prompt source gets no reference backlink

- **WHEN** the PR template is rendered for a `prompt:prompt` work item
- **THEN** the `Reference` field is empty

### Requirement: Single Build function replaces BuildDefault and BuildOverride

The prompt builder SHALL expose a single `Build` function that accepts a `workitem.WorkItem`, a `repository.Target`, and build options (override text and `--pr` flag). The `BuildDefault` and `BuildOverride` functions SHALL be removed. The `Build` function SHALL handle both the default and override cases via the options struct.

#### Scenario: Build with no override text produces default prompt

- **WHEN** `Build` is called with an empty override text and `--pr` set to false
- **THEN** the output is the same as the previous `BuildDefault` behavior (minus the PR instruction, which is now opt-in)

#### Scenario: Build with override text produces override prompt

- **WHEN** `Build` is called with override text set and `--pr` set to false
- **THEN** the output is the same as the previous `BuildOverride` behavior (minus the PR instruction, which is now opt-in)

### Requirement: Template data structure

The frame templates (`implement`, `review`) SHALL receive a data struct with the following fields: `SourceLabel` (string, the human-readable label like "GitHub issue #456", empty for `prompt:prompt`), `Context` (string, the work item title and description), `OverrideText` (string, the `--prompt` override text, empty if not set), and `PRInstruction` (string, the rendered PR template output, empty if `--pr` is not set). The PR template (`pr`) SHALL receive a data struct with a `Reference` field (string, the reference backlink note or empty).

#### Scenario: Frame template data is populated correctly

- **WHEN** `Build` is called for a `github:issue` work item with title "Fix bug" and description "The bug is in X"
- **THEN** the implement template receives `SourceLabel` = `GitHub issue #<ref>`, `Context` = `Fix bug\n\nThe bug is in X`, `OverrideText` = `""`, `PRInstruction` = `""` (when `--pr` is not set)

#### Scenario: PR template data is populated correctly

- **WHEN** the PR template is rendered for a `github:issue` work item on a `github` host
- **THEN** the PR template receives `Reference` = `, make sure the PR body references GitHub issue #<ref> with a non-closing reference like 'Refs #<ref>'`
