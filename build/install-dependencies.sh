#!/bin/bash

# Install lint tools
wget -P /usr/local/bin/hadolint https://github.com/hadolint/hadolint/releases/download/v1.17.5/hadolint-Linux-x86_64
chmod +x /usr/local/bin/hadolint

exit 0;
