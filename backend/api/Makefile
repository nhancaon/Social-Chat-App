.PHONY: clear-swag swag-init swagger

clear-swag: 
	@echo "Clearing swagger docs..."	
	del /Q docs\* 2>nul || exit 0

swag-init: clear-swag
	@echo "Generating swagger docs..."
	swag init -g main.go

# run all tests
test:
	go test ./tests/... -v

# proto contracts now live in ../protos (shared by all backend services) —
# see backend/protos/Makefile to regenerate them