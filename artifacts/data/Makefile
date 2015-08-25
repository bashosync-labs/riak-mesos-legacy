PREFIX := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
ERLC   := $(shell which erlc)

.PHONY: all clean
all: riak_explorer-bin.tar.gz riak-2.1.1-bin.tar.gz trusty.tar.gz

### Riak Explorer begin
riak_explorer:
	git clone --depth=1 https://github.com/basho-labs/riak_explorer.git
riak_explorer/rel/riak_explorer: riak_explorer OTP_R16B02_basho8-bin
	cd riak_explorer && PATH=$(PREFIX)/OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel-backend
riak_explorer-bin.tar.gz: riak_explorer/rel/riak_explorer
	tar -C riak_explorer/rel -czf riak_explorer-bin.tar.gz riak_explorer || rm -rf riak_explorer-bin.tar.gz
clean: clean_riak_explorer
clean_riak_explorer:
	-rm -rf riak_explorer-bin.tar.gz riak_explorer
### Riak Explorer end

### Riak begin
riak-2.1.1.tar.gz:
	curl -C - -O -L http://s3.amazonaws.com/downloads.basho.com/riak/2.1/2.1.1/riak-2.1.1.tar.gz
riak-2.1.1/rel/riak: riak-2.1.1.tar.gz OTP_R16B02_basho8-bin
	tar -xvf riak-2.1.1.tar.gz
	export PATH
	rm -rf riak-2.1.1/deps/node_package
	git clone https://github.com/basho/node_package.git --branch no-epmd riak-2.1.1/deps/node_package
	cd riak-2.1.1 && PATH=$(PREFIX)/OTP_R16B02_basho8-bin/bin:$(PATH) $(MAKE) rel
	touch riak-2.1.1
riak-2.1.1-bin.tar.gz: riak-2.1.1/rel/riak
	tar -C riak-2.1.1/rel/ -czf riak-2.1.1-bin.tar.gz riak || rm -rf riak-2.1.1-bin.tar.gz
clean: clean_riak
clean_riak:
	-rm -rf riak-2.1.1 riak-2.1.1-bin.tar.gz
### Riak end

### Erlang begin
ifneq ($(ERLC),)
$(shell touch otp_src_R16B02-basho8.tar.gz)
$(shell touch OTP_R16B02_basho8-bin)
endif
otp_src_R16B02-basho8.tar.gz:
	curl -C - -O http://s3.amazonaws.com/downloads.basho.com/erlang/otp_src_R16B02-basho8.tar.gz || rm otp_src_R16B02-basho8.tar.gz
OTP_R16B02_basho8-bin: otp_src_R16B02-basho8.tar.gz
	tar -xvf otp_src_R16B02-basho8.tar.gz
	cd OTP_R16B02_basho8 && ./otp_build autoconf && ./configure --prefix=$(PREFIX)/OTP_R16B02_basho8-bin && $(MAKE) && $(MAKE) install
clean: clean_erlang
clean_erlang:
	-rm -rf otp_src_R16B02-basho8.tar.gz OTP_R16B02_basho8-bin
### Erlang end

### schroot, debootstrap begin
trusty.tar.gz:
	sudo debootstrap trusty trusty_root/
	sudo chroot trusty_root apt-get install -y busybox-static strace curl wget openssl
	sudo chroot trusty_root apt-get clean
	sudo chown -R $(shell id -un): trusty_root/
	rm -rf trusty_root/dev trusty_root/proc trusty_root/sys
	mkdir -p trusty_root/dev trusty_root/proc trusty_root/sys trusty_root/parent_root
	ln -sf /parent_root/etc/resolv.conf trusty_root/etc/resolv.conf
	ln -sf /parent_root/etc/hosts trusty_root/etc/hosts
	tar -C trusty_root -czf trusty.tar.gz . || rm -rf trusty.tar.gz
	rm -rf trusty_root
clean: clean_schroot
clean_schroot:
	-rm -rf trusty.tar.gz trusty_root
### schroot, debootstrap end