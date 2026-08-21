#!/bin/bash

tmux send-keys -t %1 "$*" Enter