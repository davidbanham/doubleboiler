.PHONY: rename
rename:
	find ./ -type f | grep -v .git | xargs sed -i -e 's/doubleboiler/$(brand)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/Doubleboiler/$(titleCaseBrand)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/DOUBLEBOILER/$(upperCaseBrand)/g'

.PHONY: logos_to_paths
logos_to_paths:
	cp ./assets/img/logo_text_src.svg ./assets/img/logo_text.svg
	inkscape ./assets/img/logo_text.svg --verb EditSelectAll --verb SelectionUnGroup --verb EditSelectAll --verb ObjectToPath --verb FileSave --verb FileQuit
	scour -i assets/img/logo.svg -o assets/img/logo.min.svg
	cp ./assets/img/logo_src.svg ./assets/img/logo.svg
	inkscape ./assets/img/logo.svg --verb EditSelectAll --verb SelectionUnGroup --verb EditSelectAll --verb ObjectToPath --verb FileSave --verb FileQuit
	scour -i assets/img/logo_text.svg -o assets/img/logo_text.min.svg
	cp ./assets/img/logo_text_white_src.svg ./assets/img/logo_text_white.svg
	inkscape ./assets/img/logo_text_white.svg --verb EditSelectAll --verb SelectionUnGroup --verb EditSelectAll --verb ObjectToPath --verb FileSave --verb FileQuit
	scour -i assets/img/logo_text_white.svg -o assets/img/logo_text_white.min.svg
	convert ./assets/img/logo_text.svg ./assets/img/logo_text.png
	convert ./assets/img/logo.svg ./assets/img/logo.png
	convert ./assets/img/logo_text_white.svg ./assets/img/logo_text_white.png
	convert ./assets/img/logo.svg -resize 192x192\! ./assets/android-chrome-192x192.png
	convert ./assets/img/logo.svg -resize 512x512\! ./assets/android-chrome-512x512.png
	convert ./assets/img/logo.svg -resize 180x180\! ./assets/apple-touch-icon.png
	convert ./assets/img/logo.svg -resize 16x16\! ./assets/favicon-16x16.png
	convert ./assets/img/logo.svg -resize 32x32\! ./assets/favicon-32x32.png
	convert ./assets/img/logo.svg -resize 144x144\! ./assets/mstile-144x144.png
	convert ./assets/img/logo.svg -resize 150x150\! ./assets/mstile-150x150.png
	convert ./assets/img/logo.svg -resize 310x150\! ./assets/mstile-310x150.png
	convert ./assets/img/logo.svg -resize 310x310\! ./assets/mstile-310x310.png
	convert ./assets/img/logo.svg -resize 70x70\! ./assets/mstile-70x70.png
	convert ./assets/img/logo.svg ./assets/favicon.ico
	cp ./assets/img/logo.svg ./assets/safari-pinned-tab.svg

.PHONY: new_resource
new_resource:
	$(eval newResourceName := $(shell bash -c 'read -p "Name: " name; echo $$name'))
	$(eval titleCaseNewResourceName := $(shell awk 'BEGIN{print toupper(substr("$(newResourceName)",1,1)) substr("$(newResourceName)", 2, length("$(newResourceName)"))}'))
	for file in models/thing.go models/thing_test.go routes/things.go routes/things_test.go views/thing.html views/things.html views/create-thing.html ; do \
		cat $$file | sed 's/Thing/$(titleCaseNewResourceName)/g' \
		| sed 's/thing/$(newResourceName)/g' \
		> `echo $$file | sed 's/thing/$(newResourceName)/'` ; \
	done
	make migration migname=$(newResourceName)
