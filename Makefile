# --- Configuration Variables ---
PROJECT_ID := demo-sharetelemetry
DATA_DIR := ./emulator-data

# --- General Settings ---
.PHONY: help start dev clean-data

# Default command: Show help
help:
	@echo "----------------------------------------------------------------------"
	@echo "           🔥  SHARETELEMETRY EMULATOR MANAGER  🔥"
	@echo "----------------------------------------------------------------------"
	@echo "Available commands:"
	@echo "  make dev         - Start Firestore with data persistence (Import/Export)"
	@echo "  make start       - Start Firestore fresh (Data is lost on exit)"
	@echo "  make clean-data  - Delete the saved emulator data"
	@echo "----------------------------------------------------------------------"
	@echo "Target Project: $(PROJECT_ID)"
	@echo "----------------------------------------------------------------------"

# --- Emulator Management ---

# Recommended for Development: Loads data and saves new data on exit
dev:
	firebase emulators:start \
		--project $(PROJECT_ID) \
		--import=$(DATA_DIR) \
		--export-on-exit

# Clean Start: Starts with an empty database (good for unit testing flows)
start:
	firebase emulators:start \
		--project $(PROJECT_ID)

# Deletes the persisted data to start completely fresh next time you run 'make dev'
clean-data:
	rm -rf $(DATA_DIR)
	@echo "🗑️  Emulator data removed."