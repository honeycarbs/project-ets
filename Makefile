.PHONY: push-edu push-personal push-all

# Git push targets with credential switching
push-edu:
	@echo "Switching to educational credentials..."
	git config user.name "Tatiana Kazaeva"
	git config user.email "tkazaeva@pdx.edu"
	@echo "Pushing to CECS repository..."
	git push origin main

push-personal:
	@echo "Switching to personal credentials..."
	git config user.name "honeycarbs"
	git config user.email "honeycarbs.personal@email.com"
	@echo "Pushing final/ subtree to personal repository..."
	git subtree push --prefix=final personal main

push-all: push-edu push-personal
	@echo "Pushed to all repositories!"