# Contributing to wbn and wbn-sign packages

Follow the [main contribution guidelines](../CONTRIBUTING.md). This file
focuses on contribution instructions which are specific to the `js/` part of the
repository containing the code for `wbn` and `wbn-sign` packages.

## Auto-formatting code

The Github Actions workflow enforces linting code with Prettier according to the
Prettier configs specified in the `package.json`.

To lint your code locally before committing, one can run `npm run lint`.

To enable running Prettier on save with VSCode, one can install the Prettier
extension and then in VScode's settings have the following entries:

```json
"editor.formatOnSave": true,
"[javascript]": {
        "editor.defaultFormatter": "esbenp.prettier-vscode"
}
```

## Publishing `wbn` and `wbn-sign` to npm

Both `wbn` (`js/bundle`) and `wbn-sign` (`js/sign`) are published independently via the [`.github/workflows/publish-npm.yml`](../.github/workflows/publish-npm.yml) GitHub Actions workflow using Google's OSS Exit Gate pipeline:

1. Update the `"version"` field in `js/bundle/package.json` (for `wbn`) or `js/sign/package.json` (for `wbn-sign`) and merge the PR into `main`.
2. In GitHub, navigate to **Actions** -> **Publish to npm via OSS Exit Gate** -> **Run workflow** (on the `main` branch).
3. Select the target **`package`** (`wbn-sign` or `wbn`).
4. *(Optional)* Check **`dry_run`** to verify building, testing, Workload Identity Federation authentication, and staging to Artifact Registry without publishing to public npm. Leave `dry_run` unchecked to publish the release to npmjs.com.
