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

Each file is encrypted for two age recipients:

- the local dev key at `~/.config/sops/age/keys.txt` — **back this key up
  somewhere safe**; it is what lets you edit the secrets on this machine;
- the homelab cluster key (Flux's kustomize-controller decrypts the secrets
  in-cluster using the `infra-sops-age` secret in `flux-system`).

To edit a secret in place:

```bash
sops edit k8s/secrets/app-secrets.enc.yaml
```

After adding or removing a recipient in `.sops.yaml`, re-encrypt with
`sops updatekeys k8s/secrets/<file>.enc.yaml`.

## Deployment (GitOps via Flux)

The homelab repo (`clusters/main/apps/reminder-bot/`) points Flux at this
repo: a `GitRepository` watches `main` and a `Kustomization` applies `./k8s`
— namespace, workloads, and the sops-encrypted secrets, which Flux decrypts
in-cluster. Nothing is applied by hand; pushing to `main` is the deployment.

CD builds and pushes the image to `ghcr.io/chivta/reminder-bot` on every push
to `main`, after the build/vet/test job passes.

For a manual apply (e.g. a cluster without Flux), decrypt the secrets
yourself — `kubectl apply -k k8s/` alone would apply them still encrypted.
