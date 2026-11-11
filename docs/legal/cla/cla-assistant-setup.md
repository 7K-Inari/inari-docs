# CLA Assistant Activation Runbook

!!! danger "Prepared but disabled"

    Do not perform the activation steps until the owner and counsel approvals in [the CLA overview](README.md) are complete. No GitHub App installation, repository link, required status check, or signature request is part of the legal-hygiene implementation change.

## Prepared Configuration

- Service: [CLA assistant](https://cla-assistant.io/) using the [CLA Assistant GitHub App](https://github.com/apps/cla-assistant), app ID `128106`.
- Canonical agreements: [ICLA](icla.md) and [CCLA](ccla.md).
- Click-through custom fields: [`cla-assistant-metadata.json`](cla-assistant-metadata.json).
- Repository scope: the 11 active repositories listed below.
- Excluded: archived `7K-Inari/inari-operator`.
- Verified on 2026-11-11 through the GitHub API: the CLA Assistant app was **not installed** on `7K-Inari`.

The hosted CLA assistant service is configured through its dashboard and a GitHub Gist. It does **not** use a `.github/cla-assistant.yml` file, so no such placeholder files have been added to the repositories.

### Target Repositories

- [ ] `7K-Inari/inari-server`
- [ ] `7K-Inari/inari-agent`
- [ ] `7K-Inari/inari-docs`
- [ ] `7K-Inari/inari-ui`
- [ ] `7K-Inari/inari-release-bundle`
- [ ] `7K-Inari/inari-api`
- [ ] `7K-Inari/inari-cli`
- [ ] `7K-Inari/inari-plugin-sdk`
- [ ] `7K-Inari/inari-ui-plugin-sdk`
- [ ] `7K-Inari/inari-catalog`
- [ ] `7K-Inari/inari-ext-argocd`

## Activation Prerequisites

- [ ] Record owner and counsel approval in [the CLA overview](README.md), including the approved agreement version and governing jurisdiction.
- [ ] Merge the approved canonical agreements and all repository `CONTRIBUTING.md` files.
- [ ] Confirm that the approved CLA source commit is on `7K-Inari/inari-docs` `main`.
- [ ] Choose the 7K Group administrator account that will own the CLA Gist and the CLA assistant dashboard configuration.
- [ ] Review CLA assistant's current terms, privacy policy, signer-data storage, and export process.
- [ ] Ensure the hard gate remains closed to external contributions until validation is complete.

## Activation Steps

### 1. Publish the Click-through Agreement

1. Create a new **secret Gist** using the designated administrator account. A secret Gist is unlisted, not confidential; publish no private information in it.
2. Add `7K-Group-CLA.md` containing the approved ICLA and CCLA, clear instructions for selecting the applicable variant, the approved version, and the canonical `inari-docs` source commit SHA. Generate this file from the canonical documents rather than maintaining a separate hand-edited legal text.
3. Add a second Gist file named `metadata` containing the exact contents of [`cla-assistant-metadata.json`](cla-assistant-metadata.json).
4. Verify that both variants are readable and that the click-through page records full name, contact email, acceptance category, and explicit agreement. Corporate signers must also provide their legal entity and title or authority; review these fields because the metadata format cannot conditionally require them.
5. Record the Gist URL, approved agreement version, and canonical source commit in the activation evidence. Never commit signing records or signer personal data to the repositories.

### 2. Install and Link the App

1. Install the [CLA Assistant GitHub App](https://github.com/apps/cla-assistant) on `7K-Inari`, granting access only to the 11 target repositories.
2. In the CLA assistant dashboard, link the organization to the approved Gist if org-wide linking is available. Otherwise, link each target repository individually to the same Gist and agreement version.
3. Confirm that `inari-operator` and unrelated repositories are not linked.
4. Review automation accounts and allowlist only accounts that cannot sign and whose pull requests must remain eligible, expected to include `7k-renovate-bot[bot]` and `github-actions[bot]`. Do not allowlist human contributors to bypass the CLA.
5. Do not require the status check in branch protection yet.

### 3. Validate Before Enforcement

1. Open a disposable documentation pull request from an account with no signature and verify that CLA assistant comments and posts a non-passing status.
2. Complete the individual click-through flow and verify that the status passes and the signing record references the approved agreement version.
3. Validate the corporate flow with an authorized test signer, including legal entity and authority details.
4. Verify that a changed agreement version prompts re-signing and that allowlisted automation remains functional.
5. Export or otherwise verify access to the signing records and confirm who administers them.
6. Close the disposable pull request without merging it.

### 4. Enforce and Close the Gate

1. Require the verified CLA assistant status check for pull requests on `main` in each target repository, including merge-queue settings where applicable.
2. Update every root `CONTRIBUTING.md` to remove the pending-approval notice and record that digital sign-off is active.
3. Record the activation date, app installation ID, Gist version, validation evidence, and required-check rollout in [the CLA overview](README.md).
4. Only then mark the CLA portion of G1 complete. External contributions may proceed only after all other G1 items, including the trademark filing, are also complete.

## Rollback

To stop new CLA enforcement, unlink the affected repositories or organization in CLA assistant and remove the required-status-check setting. Suspend or uninstall the app only if necessary. Preserve approved agreement versions and signing records needed to evidence prior acceptances; do not delete records merely to bypass a failed check.
