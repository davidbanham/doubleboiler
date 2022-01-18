assets/css/main.css: views/*.html assets/css/tailwind.css
	NODE_ENV=production ./node_modules/tailwindcss/lib/cli.js -i ./assets/css/tailwind.css -o ./assets/css/main.css --jit --minify
