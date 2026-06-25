.PHONY: clear-swag swag-init swagger

clear-swag: 
	@echo "Clearing swagger docs..."	
	del /Q docs\* 2>nul || exit 0

swag-init: clear-swag
	@echo "Generating swagger docs..."
	swag init -g main.go