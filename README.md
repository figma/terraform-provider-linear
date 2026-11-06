# Terraform Linear Provider

A Terraform provider for [Linear](https://linear.app).

[Provider documentation](docs/index.md) is in this repository.

## Figma-specific documentation

### Publish a new version

**Only a Spacelift admin can run these steps.**

1. Update version number in `.goreleaser.yml`

2. The GPG key passphrase to sign plugins is in the Infra Vault in 1Password.  Search for `Foundation Team GPG Private Key - Figinfra Terraform`. You will be asked to enter it while running the command below.

3. Import GPG key to your local environment if you haven't already (requires `gnupg` and `op` CLIs):

```bash
export GPG_TTY=$(tty)
op read 'op://Infra/Foundation Team GPG Private Key - Figinfra Terraform/foundation-team.private.gpg' | gpg --import
```

4. Run `mise install` in the repo root to install `goreleaser`.

5. Run `goreleaser` to build release artifacts:
 
```bash
export GPG_FINGERPRINT='FA4BB7B3DEE23FD04930475C2E8D2A73406F181C'
export GPG_TTY=$(tty)
goreleaser release --snapshot --clean
```

6. Upload the new version to Spacelift. Copy the `VERSION_ID` from the output to use in the next step:

```bash
spacectl provider create-version --type linear --protocols 6.0 --gpg-key-id 2E8D2A73406F181C
```

7. Publish the version to make it available:

```bash
spacectl provider publish-version --version <VERSION_ID>
```
