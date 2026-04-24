RG := \rg
RG += --iglob='!alatty/gl-wrapper.h'
RG += --iglob='!*.txt'
RG += '[^a-z0-9_]5[^a-z0-9]'

INSTALL_LOC = /usr/local

current: build tar

clean_kittens:
	rm -rf build/kittens

tar:
	tar -czvf alatty.tar.gz -C linux-package .

install-tar:
	tar -xvf alatty.tar.gz --strip-components=1 -C $(INSTALL_LOC)

build:
	rm -rf linux-package
	python3 build.py

c:
	./alatty.app/Contents/MacOS/alatty

open:
	open alatty.app

size:
	-du -sh ./alatty.app
	-du -s ./alatty.app

.PHONY: build
