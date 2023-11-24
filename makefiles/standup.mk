.PHONY: recase_brand
recase_brand:
	$(eval camelCaseBrandName := $(shell go run util/cased/main.go --text "$(brand)" --format camel))
	$(eval lowerCamelCaseBrandName := $(shell go run util/cased/main.go --text "$(brand)" --format lower_camel))
	$(eval kebabCaseBrandName := $(shell go run util/cased/main.go --text "$(brand)" --format kebab))
	$(eval snakeCaseBrandName := $(shell go run util/cased/main.go --text "$(brand)" --format snake))

.PHONY: check_name
check_name: recase_brand
	@echo camel $(camelCaseBrandName)
	@echo lowerCamel $(lowerCamelCaseBrandName)
	@echo kebab $(kebabCaseBrandName)
	@echo snake $(snakeCaseBrandName)
	@echo upper $(upperCaseBrand)
	@echo lower $(upperCaseBrand)

.PHONY: rename
rename: recase_brand
	find ./ -type f | grep -v .git | xargs sed -i -e 's/DoubleBoiler/$(camelCaseBrandName)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/doubleboiler/$(lowerCaseBrand)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/doubleBoiler/$(lowerCamelCaseBrandName)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/Doubleboiler/$(camelCaseBrandName)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/double-boiler/$(kebabCaseBrandName)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/double_boiler/$(snakeCaseBrandName)/g'
	find ./ -type f | grep -v .git | xargs sed -i -e 's/DOUBLEBOILER/$(upperCaseBrand)/g'

.PHONY: logos_to_paths
logos_to_paths:
	inkscape ./assets/img/logo_text_src.svg --actions "select-all;selection-ungroup;select-all;object-to-path"  --export-filename ./assets/img/logo_text.svg
	scour -i assets/img/logo.svg -o assets/img/logo.min.svg
	inkscape ./assets/img/logo_text_white_src.svg --actions "select-all;selection-ungroup;select-all;object-to-path"  --export-filename ./assets/img/logo_text_white.svg
	scour -i assets/img/logo_text_white.svg -o assets/img/logo_text_white.min.svg
	inkscape ./assets/img/logo_src.svg --actions "select-all;selection-ungroup;select-all;object-to-path"  --export-filename ./assets/img/logo.svg
	scour -i assets/img/logo_text.svg -o assets/img/logo_text.min.svg
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
	$(eval camelCaseNewResourceName := $(shell go run util/cased/main.go --text "$(newResourceName)" --format camel))
	$(eval lowerCamelCaseNewResourceName := $(shell go run util/cased/main.go --text "$(newResourceName)" --format lower_camel))
	$(eval kebabCaseNewResourceName := $(shell go run util/cased/main.go --text "$(newResourceName)" --format kebab))
	$(eval snakeCaseNewResourceName := $(shell go run util/cased/main.go --text "$(newResourceName)" --format snake))
	for file in models/some_thing.go models/some_thing_test.go routes/some_things.go routes/some_things_test.go views/some-thing.html views/some-things.html views/create-some-thing.html ; do \
		cat $$file | sed 's/SomeThing/$(camelCaseNewResourceName)/g' \
		| sed 's/someThing/$(lowerCamelCaseNewResourceName)/g' \
		| sed 's/some-thing/$(kebabCaseNewResourceName)/g' \
		| sed 's/some_thing/$(snakeCaseNewResourceName)/g' \
		> `echo $$file | sed 's/some-thing/$(kebabCaseNewResourceName)/' | sed 's/some_thing/$(snakeCaseNewResourceName)/'` ; \
	done
	make migration migname=$(snakeCaseNewResourceName)
