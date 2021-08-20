VER?=0.0.1
MODULES=$(shell find . -mindepth 2 -maxdepth 4 -type f -name 'go.mod' | cut -c 3- | sed 's|/[^/]*$$||' | sort -u | tr / :)
targets=$(addprefix test-, $(MODULES))
root_dir=$(dir $(abspath $(lastword $(MAKEFILE_LIST))))

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all:
	$(MAKE) $(targets)

tidy-%:
	cd $(subst :,/,$*); go mod tidy && go generate ./...

fmt-%:
	cd $(subst :,/,$*); go fmt ./...

vet-%:
	cd $(subst :,/,$*); go vet ./...

test-%: tidy-% fmt-% vet-% lint-%
	cd $(subst :,/,$*); go test ./... -coverprofile cover.out

lint-%: $(GOBIN)/golangci-lint
	cd $(subst :,/,$*); golangci-lint run --path-prefix $(subst :,/,$*)/ -c ../.golangci-lint.yml ./...

release-%:
	$(eval REL_PATH=$(subst :,/,$*))
	@if ! test -f $(REL_PATH)/go.mod; then echo "Missing ./$(REL_PATH)/go.mod, terminating release process"; exit 1; fi
	git checkout main
	git pull
	git tag "$(REL_PATH)/v$(VER)"
	git push origin "$(REL_PATH)/v$(VER)"

$(GOBIN)/golangci-lint:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.41.1
