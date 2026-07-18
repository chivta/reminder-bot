#!/bin/bash

# Decrypts the sops-encrypted secrets under k8s/secrets/ and applies them
# to the cluster. Requires the age private key to be available at
# ~/.config/sops/age/keys.txt (or SOPS_AGE_KEY_FILE to point at it).

set -euo pipefail

sops -d k8s/secrets/ghcr-secret.enc.yaml | kubectl apply -f -
sops -d k8s/secrets/app-secrets.enc.yaml | kubectl apply -f -
