# Contributing to Outrig

We welcome and value contributions to Outrig! Outrig is an open source project, always open for contributors. There are several ways you can contribute:

- Submit issues related to bugs or new feature requests
- Fix outstanding [issues](https://github.com/outrigdev/outrig/issues) with the existing code
- Contribute to documentation in the `aidocs/` directory
- Spread the word on social media
- Or simply ⭐️ the repository to show your appreciation

However you choose to contribute, please be mindful and respect our [code of conduct](./CODE_OF_CONDUCT.md).

> All contributions are highly appreciated! 🥰

## Before You Start

We accept patches in the form of GitHub pull requests. If you are new to GitHub, please review this [GitHub pull request guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/proposing-changes-to-your-work-with-pull-requests/about-pull-requests).

### Contributor License Agreement

Contributions to this project must be accompanied by a Contributor License Agreement (CLA). You (or your employer) retain the copyright to your contribution, this simply gives us permission to use and redistribute your contributions as part of the project.

> On submission of your first pull request you will be prompted to sign the CLA confirming your original code contribution and that you own the intellectual property.

### Style Guide

The project uses American English.

We have a set of recommended Visual Studio Code extensions to enforce our style and quality standards. Please ensure you use these, especially [Prettier](https://prettier.io) and [EditorConfig](https://editorconfig.org), when contributing to our code.

## How to Contribute

- For minor changes, you are welcome to [open a pull request](https://github.com/outrigdev/outrig/pulls).
- For major changes, please [create an issue](https://github.com/outrigdev/outrig/issues/new) first.
- If you are looking for a place to start, take a look at issues labeled "good first issue".

### Development Environment

To build and run Outrig locally:

1. Clone the repository: `git clone https://github.com/outrigdev/outrig.git`
2. Install dependencies: `npm install`
3. Run the development server: `task dev` (automatically also runs the vite dev server)
4. Open http://localhost:5173 (the Vite port)

For more detailed information on available tasks, see the `Taskfile.yml` file.

### Create a Pull Request

Guidelines:

- Before writing any code, please look through existing PRs or issues to make sure nobody is already working on the same thing.
- Develop features on a branch - do not work on the main branch.
- For anything but minor fixes, please submit tests and documentation.
- Please reference the issue in the pull request.

## Project Structure

The project is broken into three main components:

### Frontend

Our frontend can be found in the `frontend/` directory. It is written in React TypeScript. The frontend uses Jotai for state management, with the main app state defined in `frontend/appmodel.ts`. The application uses a tab-based navigation system where the selected tab determines which component is displayed.

### Client SDK (Go)

The main library is at the project root (`outrig.go`) with additional SDK packages in `pkg/`. Data structures are in `ds.go`. Main coordination happens in `controller.go`. Various stats are collected by the collectors in `pkg/collector/*`.

### Server (Go)

Server code is in `server/`, with the entry point at `server/main-server.go`, and server-specific packages in `server/pkg/`. The server collects and processes data from the monitored application, stores it in appropriate data structures, and makes it available via RPC.

### Code Generation

TypeScript types and RPC client API are automatically generated from Go types. After modifying Go types in `pkg/rpctypes/rpctypes.go`, run `task generate` to update the TypeScript type definitions in `frontend/types/rpctypes.d.ts` and the RPC client API.

Do not manually edit generated files. Instead, modify the source Go types and run `task generate`.
