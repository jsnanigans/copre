.PHONY: diagram

diagram:
	@echo "Building diagram..."
	d2 --theme=300 --dark-theme=200 -l elk --pad 10 --sketch docs/anchoring_logic.d2
	@echo "Diagram built: docs/anchoring_logic.svg" 