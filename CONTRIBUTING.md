# Contributing Guide

## Standards

- use formatting tools like Prettier and Go fmt to maintain consistent code style.
- write clear and concise code with proper naming conventions.
- include comments and documentation where necessary to explain complex logic.
    - most of the main contributors to this project are Korean, but as an open source, please add English sentences as well.
    - distinguish between `Ko` (Korean) and `En` (English) in the comments. (e.g., `// Ko: 설명`, `// En: Explanation`)
- write unit tests/integration tests for new features and bug fixes.
- ensure all tests pass before submitting a pull request. (with `-race` flag for Go tests)
- adhere to the existing code structure and patterns used in the project.

## Branch Strategy

- Always create a dedicated branch for development.
- Once development is complete, squash merge into the `main` branch.
- After a feature or task is fully completed and no longer active, delete the branch to keep the repository clean. (or archive it if necessary)

## Issue Tracking

- If there is a related issue, always reference it in the commit message or pull request.
- Use issue references consistently (e.g. `Closes #123`, `Refs #456`).

## Commit Convention

Follow the commit message format below. (Branch names should also follow similar conventions.)

### Convention

```
<type>(<scope?>): <subject>

<body?>

<footer?>
```

- `<type>`: `feat` | `fix` | `docs` | `style` | `refactor` | `test` | `chore` | `perf` | `ci` | `build` | `revert`
- `<scope>` (Optional): modules or scopes such as `auth`, `api`, `wiki`.
- `<subject>`:
    - Use imperative sentences
    - Do not end with a period
    - Lowercase the first letter
    - Maximum 50 characters
    - English only

- `<body>` (Optional):
    - Maximum 72 characters per line
    - Focus on **What** and **Why**
    - **How** should be explained by the source code itself

- `<footer>` (Optional):
    - Issue references or related metadata

### Examples

```
feat(order): support bulk checkout
fix(payment): handle timeout correctly
refactor(auth): extract token validation logic
docs: add architecture diagram
chore: bump node version to 20.x
```

```
feat: add rate limit to login API

- prevent brute force attacks
- limit to 5 requests per minute per IP

Closes #123
Refs #456
```

## commitlint Configuration

```js
module.exports = {
    parserPreset: {
        parserOpts: {
            headerPattern: /^([a-z]+)(\([a-z0-9-]+\))?: ([a-z0-9][a-z0-9 .,/:;!?()[\]{}\-_=+#@]{0,49})$/i,
            headerCorrespondence: ['type', 'scope', 'subject'],
        },
    },

    rules: {
        'type-enum': [
            2,
            'always',
            ['feat', 'fix', 'docs', 'style', 'refactor', 'test', 'chore', 'perf', 'ci', 'build', 'revert'],
        ],
        'type-empty': [2, 'never'],
        'scope-case': [2, 'always', 'lower-case'],
        'subject-empty': [2, 'never'],
        'subject-case': [2, 'always', ['lower-case']],
        'subject-full-stop': [2, 'never', '.'],
        'subject-max-length': [2, 'always', 50],
        'body-max-line-length': [2, 'always', 72],
        'footer-max-line-length': [2, 'always', 72],
    },
}
```
