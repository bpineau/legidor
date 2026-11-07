.PHONY: build build-eval test vet install install-skill
build:
	go build -trimpath -o bin/legidor ./cmd/legidor
build-eval:
	go build -trimpath -o bin/legidor-eval ./cmd/legidor-eval
test:
	go test -race ./...
vet:
	go vet ./...
install:
	mkdir -p "$(HOME)/.local/bin"
	GOBIN="$(HOME)/.local/bin" go install ./cmd/legidor
install-skill:
	mkdir -p "$(HOME)/.agents/skills/legidor/references" "$(HOME)/.claude/skills/legidor/references"
	cp skills/legidor/SKILL.md "$(HOME)/.agents/skills/legidor/SKILL.md"
	cp skills/legidor/references/*.md "$(HOME)/.agents/skills/legidor/references/"
	cp skills/legidor/SKILL.md "$(HOME)/.claude/skills/legidor/SKILL.md"
	cp skills/legidor/references/*.md "$(HOME)/.claude/skills/legidor/references/"
