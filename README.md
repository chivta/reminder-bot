# reminder-bot

A Telegram "read-it-later" bot: send it a message and it saves it, then reminds
you about it at 09:00 and 23:00 (Europe/Kyiv) until you acknowledge it with the
"✅ Done" button.

## Local development

```bash
docker-compose up
```

Put your Telegram bot token in `bot/.env` (copy `bot/.example.env` as a
starting point). To trigger a reminder run manually without waiting for the
scheduled cron:

```bash
curl -X POST localhost:8080/remind
```

## Secrets management

Secrets are encrypted with [sops](https://github.com/getsops/sops) using an
[age](https://github.com/FiloSottile/age) key, and the encrypted files
(`k8s/secrets/*.enc.yaml`) are committed to the repo.

- The age private key lives at `~/.config/sops/age/keys.txt`. **Back this key
  up somewhere safe** — if it's lost, the secrets in this repo cannot be
  decrypted or rotated.
- To edit a secret in place:

  ```bash
  sops edit k8s/secrets/app-secrets.enc.yaml
  ```

- To apply the secrets to the currently configured cluster:

  ```bash
  ./apply-secrets.sh
  ```

## Deployment

Manifests live under `k8s/` and are applied with kustomize:

```bash
kubectl apply -k k8s/
```

Apply secrets separately (they're intentionally not part of the kustomize
resource list — see `apply-secrets.sh` above) before or after the manifests.

CD builds and pushes the image to `ghcr.io/chivta/reminder-bot` on every push
to `main`, after the build/vet/test job passes.
