<div align="center">
<h1>️  pwgo  </h1>
<h5 align="center">
Multi-list cli tool to run your Playwright suite.
</h5>
</div>
<br>
<div align="center">
  <img alt="Pwgo logo" src="./assets/pwgo-logo.png">
</div>

## Features

- 📓 New interactive selectable list view of available files, tests, and tags
- ⏳ Filterable list search
- 🔦 Tags, test and project total descriptive helpers

![Demo](./assets/pwgo-demo.gif)

#### Table of Contents

- [Installation](#installation)
- [Command line arguments](#command-line-arguments)
  - [Help mode](#help-mode)
  - [Keyboard controls](#keyboard-controls)
- [Selecting items](#selecting-items)

---

## Installation

### Homebrew

```console
brew tap dennisbergevin/tools
brew install pwgo
```

### Go

Install with Go:

```console
go install github.com/dennisbergevin/pwgo@latest
```

Or grab a binary from [the latest release](https://github.com/dennisbergevin/pwgo/releases/latest).

---

## Command line arguments

> [!NOTE]  
> For a complete list of options/arguments to pass to Playwright, refer to https://playwright.dev/docs/test-cli.

### Help mode

Common options are included in the help menu:

```bash
pwgo --help
```

![Help demo](./assets/pwgo-help.png)

### Keyboard controls

> [!NOTE]  
> All keyboard controls are displayed on the bottom of the program. Additional commands can be seen by pressing the '?' key.

|                    Keys                     |                Action                 |
| :-----------------------------------------: | :-----------------------------------: |
|               <kbd>Up/k</kbd>               |    Move to selection above current    |
|              <kbd>Down/j</kbd>              |    Move to selection below current    |
|           <kbd>Right/l/pgdn</kbd>           |   Move to next page on current list   |
|           <kbd>Left/h/pgdn</kbd>            | Move to previous page on current list |
|              <kbd>g/home</kbd>              |      Go to start of current list      |
|              <kbd>G/end</kbd>               |       Go to end of current list       |
|              <kbd>Space</kbd>               |            Select current             |
|    <kbd>Shift</kbd> + <kbd>Right/l</kbd>    |          Toggle to next list          |
|    <kbd>Shift</kbd> + <kbd>Left/h</kbd>     |        Toggle to previous list        |
|                <kbd>/</kbd>                 |          Open Filter search           |
|              <kbd>Enter</kbd>               |     Apply Filter/Run selection(s)     |
|               <kbd>Esc</kbd>                |             Remove Filter             |
| <kbd>Ctrl</kbd> + <kbd>c</kbd>/<kbd>q</kbd> |                 Quit                  |
|                <kbd>?</kbd>                 |         Open/Close help menu          |

## Selecting items

Items can be selected via the <kbd>Space</kbd> key, which will add the item to the `Selected` list.

Items can be removed from the `Selected` list and returned back to their original list via the <kbd>Space</kbd> key.

> [!NOTE]  
> If no items have been added to the `Selected` list, pressing <kbd>Enter</kbd> on an item will run that item.

![Selecting demo](./assets/pwgo-selecting.gif)
