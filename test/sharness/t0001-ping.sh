#!/usr/bin/env bash

test_description="Test ping command"

. ./sharness.sh

test_expect_success "init iptb nodes" '
  iptb init -n 2 --type=filecoin --deployment=local
'

test_expect_success "start iptb cluster" '
  iptb start
'

test_expect_success "connect iptb nodes" '
  iptb connect 0 1
  iptb connect 1 0
'

test_expect_success "get peer ids" '
  PEERID_0=$(iptb run 0 go-filecoin id | jq ".ID" | sed "s#\"##g") &&
  PEERID_1=$(iptb run 1 go-filecoin id | jq ".ID" | sed "s#\"##g")
'

test_expect_success "test ping other" '
  iptb run 0 go-filecoin ping -n2 "$PEERID_1" &&
  iptb run 1 go-filecoin ping -n2 "$PEERID_0"
'

test_expect_success "test ping self" '
   ! iptb run 0 go-filecoin ping -n2 "$PEERID_0" &&
   ! iptb run 1 go-filecoin ping -n2 "$PEERID_1"
'

test_expect_success "test ping -n0" '
  ! iptb run 0 go-filecoin ping -n0 "$PEERID_1" &&
  ! iptb run 1 go-filecoin ping -n0 "$PEERID_0"
'

test_expect_success "stop iptb" '
  iptb stop
'

test_done
