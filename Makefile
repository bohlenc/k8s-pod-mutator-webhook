IMAGE_VERSION ?= 1.0.1
IMAGE_REPO ?= docker.io/bohlenc

IMAGE_NAME_INIT ?= k8s-pod-mutator-init
IMAGE_NAME_WEBHOOK ?= k8s-pod-mutator-webhook

CGO_ENABLED ?= 0

image-init: build-image-init push-image-init

build-image-init:
	@echo "building docker image: $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION)"
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) \
    -f build/init/Dockerfile \
    --build-arg CGO_ENABLED=$(CGO_ENABLED) \
    .

push-image-init: build-image-init
	@echo "pushing docker image tags $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) and $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest"
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_INIT):$(IMAGE_VERSION)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_INIT):latest


image-webhook: build-image-webhook push-image-webhook

build-image-webhook:
	@echo "building docker image: $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION)"
	@docker build -t $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) \
  -f build/webhook/Dockerfile \
  --build-arg CGO_ENABLED=$(CGO_ENABLED) \
  .

push-image-webhook: build-image-webhook
	@echo "pushing docker image tags $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) and $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest"
	@docker tag $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION) $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):$(IMAGE_VERSION)
	@docker push $(IMAGE_REPO)/$(IMAGE_NAME_WEBHOOK):latest


.PHONY: image-init image-webhook
