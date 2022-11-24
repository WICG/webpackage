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
