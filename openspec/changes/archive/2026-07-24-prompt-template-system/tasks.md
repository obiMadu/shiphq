## 1. Template files and data structures

- [x] 1.1 Create `internal/prompt/templates/` directory with 12 embedded template files: `implement.tmpl`, `implement-github.tmpl`, `implement-gitlab.tmpl`, `implement-bitbucket.tmpl`, `review.tmpl`, `review-github.tmpl`, `review-gitlab.tmpl`, `review-bitbucket.tmpl`, `pr.tmpl`, `pr-github.tmpl`, `pr-gitlab.tmpl`, `pr-bitbucket.tmpl`. Host-specific implement/review templates match their generic counterparts. Host-specific PR templates name the concrete CLI tool (`gh`, `glab`, etc.). Generic `pr.tmpl` instructs the agent to detect the host from the git remote.
- [x] 1.2 Define `FrameData` struct (fields: `SourceLabel`, `Context`, `OverrideText`, `PRInstruction` — all strings) and `PRData` struct (field: `Reference` — string) in `internal/prompt/builder.go`.

## 2. Template resolution (TDD)

- [x] 2.1 Write `internal/prompt/resolver_test.go` — test the 5-level override hierarchy: project host-specific, global host-specific, project generic, global generic, embedded default. Test with temp dirs for project and global paths; verify the first existing file wins at each level.
- [x] 2.2 Implement `resolveTemplate(typeName, host string) (*template.Template, error)` in `internal/prompt/resolver.go` — embed `internal/prompt/templates/*.tmpl` via `//go:embed`, check disk paths in resolution order, fall back to embedded default for the type. Use `config.GetProjectConfigPath`-style project root detection (reuse `config` package or replicate the ancestor-with-`.git` walk).
- [x] 2.3 Run `go test -buildvcs=false ./internal/prompt/...` — resolver tests green.

## 3. Build function (TDD)

- [x] 3.1 Write `internal/prompt/builder_test.go` — test all spec scenarios: default implement (no --pr, no override), default review, prompt source (no label), override text on implement, override text on prompt source, --pr on implement (injects PR instruction), --pr on review (returns error), reference backlink for github:issue on github host, reference empty for github:pr and prompt:prompt.
- [x] 3.2 Implement `Build(workItem workitem.WorkItem, repoTarget repository.Target, opts BuildOptions) (string, error)` in `internal/prompt/builder.go` — resolve `SourceLabel` from work item source, resolve `Reference` from source + host, render PR template (if `opts.PRFlag` and `ModeImplement`), render frame template (`implement` or `review`) with `FrameData`, return result. If `opts.PRFlag` and `ModeReview`, return error: `--pr is only valid with implementation tasks; review tasks already inspect a PR`.
- [x] 3.3 Define `BuildOptions` struct in `internal/prompt/builder.go` with fields: `OverrideText string` and `PRFlag bool`.
- [x] 3.4 Remove `BuildDefault` and `BuildOverride` from `internal/prompt/builder.go`.
- [x] 3.5 Run `go test -buildvcs=false ./internal/prompt/...` — builder tests green.

## 4. CLI integration

- [x] 4.1 Add `prFlag bool` var and `--pr` flag registration to `main.go` `init()` on `createCmd`: `createCmd.Flags().BoolVar(&prFlag, "pr", false, "Inject PR creation instructions into the implementation prompt")`.
- [x] 4.2 Update `createCmdRun` in `main.go`: replace the `BuildDefault`/`BuildOverride` call block with a single `promptbuilder.Build(workItem, repositoryTarget, promptbuilder.BuildOptions{OverrideText: createInput.PromptOverride, PRFlag: prFlag})`.
- [x] 4.3 Run `go build -buildvcs=false` — compiles with no references to `BuildDefault` or `BuildOverride`.

## 5. Documentation

- [x] 5.1 Update `README.md` — document the `--pr` flag in the Commands and Usage sections; document the template override system (`./wtmag-prompts/` and `~/.config/wtmag/prompts/` directories, `{type}-{host}.tmpl` naming, resolution order).
- [x] 5.2 Update `config.example.toml` or add a comment block pointing users to the prompt template override directories.

## 6. Verify

- [x] 6.1 Run `go test -buildvcs=false ./...` — full suite green.
- [x] 6.2 Run `go build -buildvcs=false` — clean build, no references to `BuildDefault` or `BuildOverride`.
- [x] 6.3 Run `openspec status --change prompt-template-system` — all artifacts done, ready to verify and archive.
