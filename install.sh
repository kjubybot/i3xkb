#!/bin/bash

go install
sed -i -e "s|GOPATH|$(go env GOPATH)|" i3xkb.service
install -Dm644 i3xkb.service $HOME/.config/systemd/user/
