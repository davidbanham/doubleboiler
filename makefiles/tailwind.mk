assets/css/main.css: views/*.html assets/css/tailwind.css
	NODE_ENV=production npx tailwindcss --input ./assets/css/custom.css --output ./assets/css/main.css --minify

.PHONY: tailwind_watcher
tailwind_watcher:
	NODE_ENV=production npx tailwindcss --input ./assets/css/custom.css --output ./assets/css/main.css --minify --watch
