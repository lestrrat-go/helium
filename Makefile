INTERNAL_BIN_DIR=_internal_bin
GOVERSION=$(shell go version)
GOOS=$(word 1,$(subst /, ,$(lastword $(GOVERSION))))
GOARCH=$(word 2,$(subst /, ,$(lastword $(GOVERSION))))
HAVE_GLIDE:=$(shell which glide > /dev/null 2>1)

glide: $(IINTERNAL_BIN_DIR)/$(GOOS)/$(GOARCH)/glide

$(IINTERNAL_BIN_DIR)/$(GOOS)/$(GOARCH)/glide:
ifndef HAVE_GLIDE
	@echo "Installing glide for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(INTERNAL_BIN_DIR)/$(GOOS)/$(GOARCH)
	@wget -q -O - https://github.com/Masterminds/glide/releases/download/0.12.3/glide-0.12.3-$(GOOS)-$(GOARCH).tar.gz | tar xvz
	@mv $(GOOS)-$(GOARCH)/glide $(INTERNAL_BIN_DIR)/$(GOOS)/$(GOARCH)/glide
	@rm -rf $(GOOS)-$(GOARCH)
endif

installdeps:
	@PATH=$(INTERNAL_BIN_DIR)/$(GOOS)/$(GOARCH):$(PATH) glide install
