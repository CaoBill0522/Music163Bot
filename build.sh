#!/bin/bash
set -e

CGO_ENABLED=0 go build -trimpath -o Music163bot-Go .
