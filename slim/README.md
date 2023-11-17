# slim: Basic Dagger Module (slim "build" command)

Dagger module to minify the target container image.

This is a fork of the original code from https://github.com/shykes/daggerverse/tree/main/slim


## `minify` Function/Command Flags

* `--container CONTAINER_IMAGE_NAME` - target container image (required parameter)
* `--mode` - the engine execution mode `docker` (containerized version using Docker service) or `native` (native Dagger execution mode) (`docker` by default, optional)
* `--probe-http true|false` - enable HTTP probing of the temporary container (enabled by default, optional)
* `--probe-exec SHELL_CMD_TO_RUN` - enable exec-based probing of the temporary container (optional)
* `--show-clogs true|false` - show temporary container logs (optional)
* `--slim-debug true|false` - enable debug output in SlimToolkit (optional)


## Demo Steps (from the module itself)

Call the `minify` module function to minify the target image and expose its network port when the minified image is executed at end:

`dagger up --port 8080:80 minify --container nginx:latest`

Save the minified image as a tar file:

`dagger download --output ./nginx-slim.tar minify --container nginx:latest`

Load the minified image from the saved tar file:

`docker load -i ./nginx-slim.tar`

The `docker load` command will print `Loaded image ID: YOUR_IMAGE_HASH`

Tag the loaded image, so it's easier to use later:

`docker tag YOUR_IMAGE_HASH nginx-slim:latest`

Run the minified container image in your host environment:

`docker run -it --rm -p 8888:80 nginx-slim:latest`


# Notes

The examples repo has many minification examples for different application stacks and different base images: https://github.com/slimtoolkit/examples

See the main repo to get more information about the available flags (look for the `build` command flags): https://github.com/slimtoolkit/slim