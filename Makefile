TAG ?=

IMAGE_VERSION ?= 1.1.0
IMAGE_REPO ?= docker.io/bohlenc

IMAGE_NAME_INIT ?= k8s-pod-mutator-init
IMAGE_NAME_WEBHOOK ?= k8s-pod-mutator-webhook

CGO_ENABLED ?= 0

image-init: docker-build-init docker-push-init

docker-build-init:
	@echo "building docker image: $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION)"
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) \
    -f build/init/Dockerfile \
    --build-arg CGO_ENABLED=$(CGO_ENABLED) \
    .

docker-push-init: docker-build-init
	@echo "pushing docker image tags $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) and $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest"
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest


image-webhook: docker-build-webhook docker-push-webhook

docker-build-webhook:
	@echo "building docker image: $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION)"
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) \
  -f build/webhook/Dockerfile \
  --build-arg CGO_ENABLED=$(CGO_ENABLED) \
  .

docker-push-webhook: docker-build-webhook
	@echo "pushing docker image tags $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) and $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest"
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest


tag: set-version git-push-new-version
	@echo "creating tag $(TAG)"
	@git tag $(TAG)
	@git push origin $(TAG)

set-version:
	@echo "updating appVersion in deploy/helm/Chart.yaml"
	@sed -i "" "s/appVersion: [[:digit:]].[[:digit:]].[[:digit:]]/appVersion: $(TAG)/g" deploy/helm/Chart.yaml
	@echo "updating IMAGE_VERSION in Makefile"
	@sed -i "" "s/IMAGE_VERSION ?= [[:digit:]].[[:digit:]].[[:digit:]]/IMAGE_VERSION ?= $(TAG)/g" Makefile

git-push-new-version:
	@git add --all
	@git commit -m "[tag] $(TAG)"
	@git push

.PHONY: image-init image-webhook tag
