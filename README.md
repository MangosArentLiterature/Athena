![Athena logo](resource/logo.png)<br>
Athena is a lightweight AO2 server written in Go.<br>
Athena was created with a few core principles in mind:
* Being fast and efficient: Athena is built on concurrency, leveraging the full power of modern multi-core CPUs.
* Being simple to setup and configure.
* Having a more minimalist feature list, retaining vital and often used features while discarding unnessecary bloat.

## Features
* WebAO support
* Concurrent handling of client connections
* A moderator user system with configurable roles to set permissions
* A robust command system
* Easy to understand configuration using [TOML](https://toml.io/en/)
* Passwords stored using bcrypt
* A CLI command parser, allowing basic commands to be run without connecting with a client
* A privacy-oriented logging system, allowing for easy moderation while maintaining user privacy
* Testimony recorder

## Quick Start
Download the [latest release](https://github.com/MangosArentLiterature/Athena/releases/latest), extract into a folder of your chosing.<br>
Rename `config_sample` to `config` and modify the configuration files.<br>
Run the executable and setup your initial moderator account with `mkusr`.

## Configuration
By default, athena looks for its configuration files in the `config` directory.<br>
If you'd like to store your configuration files elsewhere, you can pass the `-c` flag on startup with the path to your configuration directory.<br>
CLI input can be disabled with `-nocli`