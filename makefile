SHELL := /bin/bash

# ==============================================================================
# Testing running system

run:
	go run main.go

# ==============================================================================
# Building containers

VERSION := 1.0

all: strava

strava:
	docker build \
		-f zarf/docker/dockerfile.strava \
		-t strava-amd64:$(VERSION) \
		.

# ==============================================================================
# Running from within k8s/kind

KIND_CLUSTER := diaz-starter-cluster

# Upgrade to latest Kind (>=v0.11): e.g. brew upgrade kind
# For full Kind v0.11 release notes: https://github.com/kubernetes-sigs/kind/releases/tag/v0.11.0
# Kind release used for our project: https://github.co	m/kubernetes-sigs/kind/releases/tag/v0.11.1
# The image used below was copied by the above link and supports both amd64 and arm64.

kind-up:
	kind create cluster \
		--image kindest/node:v1.22.0@sha256:b8bda84bb3a190e6e028b1760d277454a72267a5454b57db34437c34a588d047 \
		--name $(KIND_CLUSTER) \
		--config zarf/k8s/kind/kind-config.yaml

kind-down:
	kind delete cluster --name $(KIND_CLUSTER)

kind-load:
	kind load docker-image strava-amd64:$(VERSION) envoyproxy/envoy-dev:latest --name $(KIND_CLUSTER)

kind-apply:
	cat zarf/k8s/kind/strava-envoy/strava-envoy.yaml | kubectl apply -f -

kind-delete:
	cat zarf/k8s/kind/strava-envoy/strava-envoy.yaml | kubectl delete -f -

kind-status:
	kubectl get nodes -o wide
	kubectl get svc -o wide
	kubectl get pods -o wide --watch --all-namespaces

kind-status-strava:
	watch "kubectl get pods -o wide"

# Logging support currently working. Need to find how to get $(kubctl ...) output to work in makefile
kind-strava-logs:
	kubectl logs $(kubectl get pods --selector=app=strava -o jsonpath='{.items..metadata.name}') -c strava -f

kind-envoy-logs:
	kubectl logs $(kubectl get pods --selector=app=strava -o jsonpath='{.items..metadata.name}') -c envoy -f

kind-restart:
	kubectl rollout restart deployment strava-pod

kind-update: all kind-load kind-restart

kind-update-apply: all kind-load kind-apply

# ==============================================================================
# Modules support

tidy:
	go mod tidy