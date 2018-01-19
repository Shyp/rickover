SHELL = /bin/bash -o pipefail

BAZEL_VERSION := 0.6.1
BAZEL_DEB := bazel_$(BAZEL_VERSION)_amd64.deb

BUMP_VERSION := $(GOPATH)/bin/bump_version
MEGACHECK := $(GOPATH)/bin/megacheck

test: vet
	bazel test \
		--experimental_repository_cache="$$HOME/.bzrepos" \
		--test_output=errors //...

vet: | $(MEGACHECK)
	go vet ./...
	$(MEGACHECK) ./...

$(MEGACHECK):
	go get honnef.co/go/tools/cmd/megacheck

install-travis:
	wget "https://storage.googleapis.com/bazel-apt/pool/jdk1.8/b/bazel/$(BAZEL_DEB)"
	sudo dpkg --force-all -i $(BAZEL_DEB)
	sudo apt-get install moreutils -y

race-test: vet
	bazel test \
		--experimental_repository_cache="$$HOME/.bzrepos" \
		--spawn_strategy=remote \
		--strategy=Closure=remote \
		--strategy=Javac=remote \
		--test_output=errors --features=race //...

ci:
	bazel --batch --host_jvm_args=-Dbazel.DigestFunction=SHA1 test \
		--experimental_repository_cache="$$HOME/.bzrepos" \
		--spawn_strategy=remote \
		--test_output=errors \
		--strategy=Javac=remote \
		--profile=profile.out \
		--noshow_progress \
		--noshow_loading_progress \
		--features=race //... 2>&1 | ts '[%Y-%m-%d %H:%M:%.S]'
	bazel analyze-profile --curses=no --noshow_progress profile.out

$(BUMP_VERSION):
	go get github.com/Shyp/bump_version

release: race-test | $(BUMP_VERSION)
	$(BUMP_VERSION) minor client.go
