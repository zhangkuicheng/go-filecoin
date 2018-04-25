#!/usr/bin/env bash

test_description="Test mining command"

. ./sharness.sh

test_expect_success "init iptb nodes" '
  NUM_NODES=50
  iptb init -n "$NUM_NODES" --type=filecoin --deployment=local
'

test_expect_success "start iptb cluster" '
  iptb start
'

test_expect_success "connect iptb nodes" '
  iptb connect [0-49] [0-49]
'

test_expect_success "each node mines a block" '
  for ((i=0; i<"$NUM_NODES"; i++))
  do
    iptb run "$i" go-filecoin mining once
    sleep 1
  done
'

test_expect_success "block chain lenght is correct" '
  EXPECT=`expr $NUM_NODES + 1`
  echo "$EXPECT" > expect
  for ((i=0; i<"$NUM_NODES"; i++))
  do
    iptb run "$i" go-filecoin chain ls --enc=json | tee debug"$i"  |wc -l > actual
    test_cmp actual expect
  done
'

test_expect_success "stop iptb" '
  iptb stop
'

test_done
