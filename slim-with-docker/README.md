# slim-with-docker: Basic Dagger Module (slim "build" command)

Dagger module to minify the target container image using containerized version of SlimToolkit and a Docker service.

This is a fork of the original code from https://github.com/shykes/daggerverse/tree/main/slim

## Flags

* `--container CONTAINER_IMAGE_NAME` - target container image (required parameter)
* `--probe-http true|false` - enable HTTP probing of the temporary container (enabled by default, optional)
* `--probe-exec SHELL_CMD_TO_RUN` - enable exec-based probing of the temporary container (optional)
* `--show-clogs true|false` - show temporary container logs (optional)
* `--slim-debug true|false` - enable debug output in SlimToolkit (optional)
