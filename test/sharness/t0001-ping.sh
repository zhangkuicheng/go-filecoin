#!/usr/bin/env bash

test_description="Test ping command"

. ./sharness.sh

test_expect_success "init iptb nodes" '
  iptb-private init -n 2 --type=filecoin --deployment=local
'

test_expect_success "start iptb cluster" '
  iptb-private start
'

test_expect_success "connect iptb nodes" '
  iptb-private connect 0 1
  iptb-private connect 1 0
'

test_expect_success "get peer ids" '
  PEERID_0=$(iptb-private run 0 go-filecoin id | jq ".ID" | sed "s#\"##g") &&
  PEERID_1=$(iptb-private run 1 go-filecoin id | jq ".ID" | sed "s#\"##g")
'

test_expect_success "test ping other" '
  iptb-private run 0 go-filecoin ping -n2 "$PEERID_1" &&
  iptb-private run 1 go-filecoin ping -n2 "$PEERID_0"
'

test_expect_success "test ping self" '
   ! iptb-private run 0 go-filecoin ping -n2 "$PEERID_0" &&
   ! iptb-private run 1 go-filecoin ping -n2 "$PEERID_1"
'

test_expect_success "test ping -n0" '
  ! iptb-private run 0 go-filecoin ping -n0 "$PEERID_1" &&
  ! iptb-private run 1 go-filecoin ping -n0 "$PEERID_0"
'

test_expect_success "stop iptb" '
  iptb-private stop
'

test_done
