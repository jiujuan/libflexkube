# Build parameters.
CGO_ENABLED=0
LD_FLAGS="-extldflags '-static'"

# Go parameters.
GOCMD=env GO111MODULE=on CGO_ENABLED=$(CGO_ENABLED) go
GOTEST=$(GOCMD) test -covermode=atomic -buildmode=exe
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GORUN=$(GOCMD) run
GOBUILD=$(GOCMD) build -v -buildmode=exe -ldflags $(LD_FLAGS)

CC_TEST_REPORTER_ID=6e107e510c5479f40b0ce9166a254f3f1ee0bc547b3e48281bada1a5a32bb56d
GOLANGCI_LINT_VERSION=v1.32.1
BIN_PATH=$$HOME/bin

GO_PACKAGES=./...
GO_TESTS=^.*$

INTEGRATION_IMAGE=flexkube/libflexkube-integration

INTEGRATION_CMD=docker run -it --rm -v /run:/run -v /home/core/libflexkube:/usr/src/libflexkube -v /home/core/go:/go -v /home/core/.password:/home/core/.password -v /home/core/.ssh:/home/core/.ssh -v /home/core/.cache:/root/.cache -w /usr/src/libflexkube --net host $(INTEGRATION_IMAGE)

E2E_IMAGE=flexkube/libflexkube-e2e

E2E_CMD=docker run -it --rm -v /home/core/libflexkube:/root/libflexkube -v /home/core/.ssh:/root/.ssh -w /root/libflexkube --net host --entrypoint /bin/bash -e TF_VAR_flatcar_channel=$(FLATCAR_CHANNEL) -e TF_VAR_controllers_count=$(CONTROLLERS) -e TF_VAR_workers_count=$(WORKERS) -e TF_VAR_nodes_cidr=$(NODES_CIDR) $(E2E_IMAGE)

BUILD_CMD=docker run -it --rm -v /home/core/libflexkube:/usr/src/libflexkube -v /home/core/go:/go -v /home/core/.cache:/root/.cache -v /run:/run -w /usr/src/libflexkube $(INTEGRATION_IMAGE)

BINARY_IMAGE=flexkube/libflexkube

# godox            - Triggers on TODOs in the code, which is fine to put.
# lll              - As some lines are long because of the type names, and breaking it down decreases redability.
# testpackage      - Disabled until tests are splitted and moved to the right file names.
# goerr113         - Disabled until we implement some error types and migrate to use them.
# gci              - As we use formatting rules from different linter and they are conflicting.
# exhaustivestruct - To be able to make use of Go zero-value feature.
DISABLED_LINTERS=godox,lll,testpackage,goerr113,gci,exhaustivestruct

TERRAFORM_BIN=$(TERRAFORM_ENV) /usr/bin/terraform

CONTROLLERS=$(shell (grep CONTROLLERS .env 2>/dev/null || echo "1") | cut -d= -f2 2>/dev/null)

WORKERS=$(shell (grep WORKERS .env 2>/dev/null || echo "2") | cut -d= -f2 2>/dev/null)

NODES_CIDR=$(shell (grep NODES_CIDR .env 2>/dev/null || echo "192.168.50.0/24") | cut -d= -f2 2>/dev/null)

FLATCAR_CHANNEL=$(shell (grep FLATCAR_CHANNEL .env 2>/dev/null || echo "stable") | cut -d= -f2 2>/dev/null)

TERRAFORM_ENV=TF_VAR_flatcar_channel=$(FLATCAR_CHANNEL) TF_VAR_controllers_count=$(CONTROLLERS) TF_VAR_workers_count=$(WORKERS) TF_VAR_nodes_cidr=$(NODES_CIDR)

VAGRANTCMD=$(TERRAFORM_ENV) vagrant

.PHONY: all
all: build build-test test lint

.PHONY: all-cover
all-cover: build build-test test-cover lint

.PHONY: build
build:
	$(GOBUILD) ./cmd/...

.PHONY: build-bin
build-bin:
	mkdir -p ./bin
	cd bin && for i in $$(ls ../cmd); do $(GOBUILD) ../cmd/$$i; done

.PHONY: build-docker
build-docker:
	docker build -t $(BINARY_IMAGE) .

.PHONY: build-e2e
build-e2e:
	docker build -t $(E2E_IMAGE) e2e

.PHONY: build-test
build-test:
	$(GOTEST) -run=nope -tags integration,e2e $(GO_PACKAGES)

.PHONY: clean
clean:
	rm -r ./bin c.out coverage.txt kubeconfig local-testing/resources local-testing/values local-testing/state.yaml 2>/dev/null || true
	make vagrant-destroy || true

.PHONY: test
test: build-test
	$(GOTEST) -run $(GO_TESTS) $(GO_PACKAGES)

.PHONY: download
download:
	$(GOMOD) download

.PHONY: test-race
test-race: build-test
	$(GOTEST) -run $(GO_TESTS) -race $(GO_PACKAGES)

.PHONY: test-integration
test-integration: build-test
	$(GOTEST) -run $(GO_TESTS) -tags=integration $(GO_PACKAGES)

.PHONY: test-cover
test-cover: build-test
	$(GOTEST) -run $(GO_TESTS) -coverprofile=$(PROFILEFILE) $(GO_PACKAGES)

.PHONY: test-mutate
test-mutate: install-go-mutesting
	go-mutesting $(GO_PACKAGES)

.PHONY: cover-browse
cover-browse:
	go tool cover -html=$(PROFILEFILE)

.PHONY: test-cover-browse
test-cover-browse: PROFILEFILE=c.out
test-cover-browse: test-cover cover-browse

.PHONY: test-e2e-run
test-e2e-run:
	helm repo update
	env $(TERRAFORM_ENV) $(GOTEST) -v -tags e2e ./e2e/

.PHONY: test-e2e
test-e2e: test-e2e-run

.PHONY: test-local-apply
test-local-apply:
	env $(TERRAFORM_ENV) $(GOTEST) -v -tags e2e ./local-testing/

.PHONY: test-conformance
test-conformance:SHELL=/bin/bash
test-conformance:
	until kubectl get nodes >/dev/null; do sleep 1; done
	sonobuoy run --mode=certified-conformance || true
	until sonobuoy status | grep e2e | grep complete; do timeout --foreground 10m sonobuoy logs -f || true; sleep 1; done
	sonobuoy results $$(sonobuoy retrieve)

.PHONY: test-conformance-clean
test-conformance-clean:
	sonobuoy delete

.PHONY: lint
lint:
	golangci-lint run --enable-all --disable=$(DISABLED_LINTERS) --max-same-issues=0 --max-issues-per-linter=0 --build-tags integration,e2e --timeout 10m --exclude-use-default=false $(GO_PACKAGES)

.PHONY: update
update:
	$(GOGET) -u $(GO_PACKAGES)
	$(GOMOD) tidy

.PHONY: codespell
codespell:
	codespell -S .git,state.yaml,go.sum,terraform.tfstate,terraform.tfstate.backup

.PHONY: codespell-pr
codespell-pr:
	git diff master..HEAD | grep -v ^- | codespell -
	git log master..HEAD | codespell -

.PHONY: format
format:
	goimports -l -w $$(find . -name '*.go' | grep -v '^./vendor')

.PHONY: codecov
codecov: PROFILEFILE=coverage.txt
codecov: SHELL=/bin/bash
codecov: test-cover
codecov:
	bash <(curl -s https://codecov.io/bash)

.PHONY: codeclimate-prepare
codeclimate-prepare:
	cc-test-reporter before-build

.PHONY: codeclimate
codeclimate: PROFILEFILE=c.out
codeclimate: codeclimate-prepare test-cover
codeclimate:
	env CC_TEST_REPORTER_ID=$(CC_TEST_REPORTER_ID) cc-test-reporter after-build --exit-code $(EXIT_CODE)

.PHONY: cover-upload
cover-upload: codecov
	# Make codeclimate as command, as we need to run test-cover twice and make deduplicates that.
	# Go test results are cached anyway, so it's fine to run it multiple times.
	make codeclimate

.PHONY: install-golangci-lint
install-golangci-lint:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(BIN_PATH) $(GOLANGCI_LINT_VERSION)

.PHONY: install-cc-test-reporter
install-cc-test-reporter:
	curl -L https://codeclimate.com/downloads/test-reporter/test-reporter-latest-linux-amd64 > $(BIN_PATH)/cc-test-reporter
	chmod +x $(BIN_PATH)/cc-test-reporter

.PHONY: install-ci
install-ci: install-golangci-lint install-cc-test-reporter

.PHONY: install-go-mutesting
install-go-mutesting:
	GO111MODULE=off go get github.com/AntonStoeckl/go-mutesting/cmd/go-mutesting

.PHONY: vagrant-up
vagrant-up:
	$(VAGRANTCMD) up

.PHONY: vagrant-rsync
vagrant-rsync:
	$(VAGRANTCMD) rsync

.PHONY: vagrant-destroy
vagrant-destroy:
	$(VAGRANTCMD) destroy --force

.PHONY: vagrant
vagrant: SHELL=/bin/bash
vagrant:
	alias vagrant='$(VAGRANTCMD)'

.PHONY: vagrant-integration-build
vagrant-integration-build:
	$(VAGRANTCMD) ssh -c "docker build -t $(INTEGRATION_IMAGE) libflexkube/integration"

.PHONY: vagrant-integration-run
vagrant-integration-run:
	$(VAGRANTCMD) ssh -c "$(INTEGRATION_CMD) make test-integration GO_PACKAGES=$(GO_PACKAGES)"

.PHONY: vagrant-integration-shell
vagrant-integration-shell:
	$(VAGRANTCMD) ssh -c "$(INTEGRATION_CMD) bash"

.PHONY: vagrant-integration
vagrant-integration: CONTROLLERS=1
vagrant-integration: WORKERS=0
vagrant-integration: vagrant-up vagrant-rsync vagrant-integration-build vagrant-integration-run

.PHONY: vagrant-build-bin
vagrant-build-bin: vagrant-integration-build
	$(VAGRANTCMD) ssh -c "$(BUILD_CMD) make build-bin"

.PHONY: vagrant-e2e-build
vagrant-e2e-build:
	$(VAGRANTCMD) ssh -c "$(BUILD_CMD) make build-e2e"

.PHONY: vagrant-e2e-kubeconfig
vagrant-e2e-kubeconfig:
	scp -P 2222 -i ~/.vagrant.d/insecure_private_key core@127.0.0.1:/home/core/libflexkube/e2e/kubeconfig ./e2e/kubeconfig

.PHONY: vagrant-e2e-run
vagrant-e2e-run: vagrant-up vagrant-rsync vagrant-build-bin vagrant-e2e-build
	$(VAGRANTCMD) ssh -c "$(E2E_CMD) -c 'make test-e2e-run'"
	make vagrant-e2e-kubeconfig

.PHONY: vagrant-e2e-destroy
vagrant-e2e-destroy:
	$(VAGRANTCMD) ssh -c "$(E2E_CMD) -c 'make test-e2e-destroy'"

.PHONY: vagrant-e2e-shell
vagrant-e2e-shell:
	$(VAGRANTCMD) ssh -c "$(E2E_CMD)"

.PHONY: vagrant-e2e
vagrant-e2e: vagrant-e2e-run vagrant-e2e-destroy vagrant-destroy

.PHONY: vagrant-conformance-run
vagrant-conformance-run:
	# Make sure static controlplane is shut down.
	$(VAGRANTCMD) ssh -c "docker stop kube-apiserver kube-scheduler kube-controller-manager"
	# Wait leaseDurationSeconds to make sure self-hosted kube-scheduler and kube-controller-manager takes over.
	sleep 15
	$(VAGRANTCMD) ssh -c "$(E2E_CMD) -c 'make test-conformance'"

.PHONY: vagrant-conformance
vagrant-conformance: vagrant-e2e-run vagrant-conformance-run vagrant-conformance-copy-results

.PHONY: vagrant-conformance-copy-results
vagrant-conformance-copy-results:
	scp -P 2222 -i ~/.vagrant.d/insecure_private_key core@127.0.0.1:/home/core/libflexkube/*.tar.gz ./

.PHONY: libvirt-apply
libvirt-apply: libvirt-download-image
	cd libvirt && $(TERRAFORM_BIN) init && $(TERRAFORM_BIN) apply -auto-approve

.PHONY: libvirt-destroy
libvirt-destroy:
	cd libvirt && $(TERRAFORM_BIN) init && $(TERRAFORM_BIN) destroy -auto-approve

.PHONY: libvirt-download-image
libvirt-download-image:
	((test -f libvirt/flatcar_production_qemu_image.img.bz2 || test -f libvirt/flatcar_production_qemu_image.img) || wget https://$(FLATCAR_CHANNEL).release.flatcar-linux.net/amd64-usr/current/flatcar_production_qemu_image.img.bz2 -O libvirt/flatcar_production_qemu_image.img.bz2) || true
	(test -f libvirt/flatcar_production_qemu_image.img.bz2 && bunzip2 libvirt/flatcar_production_qemu_image.img.bz2 && rm libvirt/flatcar_production_qemu_image.img.bz2) || true
	qemu-img resize libvirt/flatcar_production_qemu_image.img +5G

.PHONY: test-static
test-static:
	$(GORUN) honnef.co/go/tools/cmd/staticcheck $(GO_PACKAGES)

.PHONY: terraform-fmt
terraform-fmt:
	for j in $$(for i in $$(find -name *.tf 2>/dev/null | grep -v .terraform); do dirname $$i; done | sort  | uniq); do terraform fmt -check $$j; done
