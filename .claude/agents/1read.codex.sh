#!/bin/bash

tmux capture-pane -t %1 -p | tail -100