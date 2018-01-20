#!/bin/sh
#
# Copyright (c) 2018 Whyrusleeping
# MIT Licensed; see the LICENSE file in this repository.
#

test_description="Test basic filecoin daemon activities"

. lib/test-lib.sh

mkdir -p "$FIL_PATH"

test_launch_filecoin_daemon

sleep 2

test_expect_success "create a miner" '
	MINER=$(go-filecoin miner new 150000)
	echo "Miner is $MINER"
'

test_expect_success "add an ask" '
	go-filecoin order ask add "$MINER" 150 8000
'

test_expect_success "add a bid" '
	go-filecoin order bid add 150 4000
'

test_expect_success "lets make a deal" '
	go-filecoin order deal make $MINER 0 0
'

test_kill_filecoin_daemon

test_done
