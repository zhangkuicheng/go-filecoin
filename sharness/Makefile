# Run sharness tests
#
# Copyright (c) 2014 Christian Couder
# MIT Licensed; see the LICENSE file in this repository.
#
# NOTE: run with TEST_VERBOSE=1 for verbose sharness tests.

T = $(sort $(wildcard t[0-9][0-9][0-9][0-9]-*.sh))
LIBDIR = lib
SHARNESSDIR = sharness
AGGREGATE = $(LIBDIR)/$(SHARNESSDIR)/aggregate-results.sh

# Binaries generated
BINS = bin/go-filecoin bin/go-sleep

# Source files location
GO_FILECOIN_SRC = ../

GO_SLEEP_SRC = ./deps/go-sleep

# User might want to override those on the command line
GOFLAGS =

all: aggregate

clean: clean-test-results
	@echo "*** $@ ***"
	-rm -rf $(BINS)

clean-test-results:
	@echo "*** $@ ***"
	-rm -rf test-results

$(T): clean-test-results deps
	@echo "*** $@ ***"
	./$@

aggregate: clean-test-results $(T)
	@echo "*** $@ ***"
	ls test-results/t*-*.sh.*.counts | $(AGGREGATE)

# Needed dependencies.
deps: sharness $(BINS)

sharness:
	@echo "*** checking $@ ***"
	lib/install-sharness.sh

find_go_files = $(shell find $(1) -name "*.go")

bin/go-filecoin: $(call find_go_files, $(GO_FILECOIN_SRC)) BUILD-OPTIONS
	@echo "*** installing $@ ***"
	go build $(GOFLAGS) -o $@ $(GO_FILECOIN_SRC)

bin/go-sleep:
	@echo "*** installing $@ ***"
	go build $(GOFLAGS) -o $@ $(GO_SLEEP_SRC)


BUILD-OPTIONS: FORCE
	@bin/checkflags '$@' '$(GOFLAGS)' '*** new Go flags ***'

.PHONY: all clean clean-test-results $(T) aggregate deps sharness FORCE

