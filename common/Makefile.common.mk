#!/bin/bash
#
# Copyright 2022 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

############################################################
# GKE section
############################################################
PROJECT ?= oceanic-guard-191815
ZONE    ?= us-east5-c
CLUSTER ?= bedrock-prow

CONTROLLER_GEN ?= $(shell which controller-gen)

activate-serviceaccount:
ifdef GOOGLE_APPLICATION_CREDENTIALS
	gcloud auth activate-service-account --key-file="$(GOOGLE_APPLICATION_CREDENTIALS)"
endif

get-cluster-credentials: activate-serviceaccount
	gcloud container clusters get-credentials "$(CLUSTER)" --project="$(PROJECT)" --zone="$(ZONE)"

config-docker: get-cluster-credentials
	@common/scripts/config_docker.sh

lint-copyright-banner:
	@${FINDFILES} \( -name '*.go' -o -name '*.cc' -o -name '*.h' -o -name '*.proto' -o -name '*.py' -o -name '*.sh' \) \( ! \( -name '*.gen.go' -o -name '*.pb.go' -o -name '*_pb2.py' \) \) -print0 |\
		${XARGS} common/scripts/lint_copyright_banner.sh

lint-go:
	@echo go lint
	# @${FINDFILES} -name '*.go' \( ! \( -name '*.gen.go' -o -name '*.pb.go' \) \) -print0 | ${XARGS} common/scripts/lint_go.sh

lint-all: lint-go

# Run go vet for this project. More info: https://golang.org/cmd/vet/
code-vet:
	@echo go vet
	go vet $$(go list ./... )

# Run go fmt for this project
code-fmt:
	@echo go fmt
	go fmt $$(go list ./... )

# Run go mod tidy to update dependencies
code-tidy:
	@echo go mod tidy
	go mod tidy -v

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(CONTROLLER_GEN))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	GO111MODULE=on go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
endif

code-gen: ## Generate code e.g. API etc.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

bundle:
	@common/scripts/create_bundle.sh ${CSV_VERSION}

.PHONY: code-vet code-fmt code-tidy code-gen csv-gen lint-copyright-banner lint-go lint-all config-docker bundle generate
