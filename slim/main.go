package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

type Slim struct{}

func (s *Slim) Nop(ctx context.Context,
	name string,
	doit Optional[bool],
) (string, error) {
	return fmt.Sprintf("nop(name=%s,doit=%v)", name, doit.GetOr(true)), nil
}

func (s *Slim) Debug(ctx context.Context, container *Container) (*Container, error) {
	slimmed, err := s.Minify(ctx,
		container,
		OptEmpty[string](),
		OptEmpty[bool](),
		OptEmpty[string](),
		OptEmpty[bool](),
		OptEmpty[bool]())
	if err != nil {
		return nil, err
	}
	
	debug := dag.
		Container().
		From("alpine").
		WithMountedDirectory("/slim", slimmed.Rootfs()).
		WithMountedDirectory("/unslim", container.Rootfs())
	return debug, nil
}

const (
	//todo: multi-arch engine image
	engineImageARM = "index.docker.io/dslim/slim-arm"
	engineImageAMD = "index.docker.io/dslim/slim"
	archAMD64      = "amd64"
	archARM64      = "arm64"

	outputImageTag = "slim-output:latest"
	outputImageTar = "output.tar"

	flagDebug     = "--debug"
	trueValue     = "true"
	cmdBuild      = "build"
	flagShowClogs = "--show-clogs"
	flagHttpProbe = "--http-probe"
	flagExecProbe = "--exec"

	modeDocker = "docker"
	modeNative = "native"
)

func engineImage() string {
	switch runtime.GOARCH {
	case archAMD64:
		return engineImageAMD
	case archARM64:
		return engineImageARM
	default:
		return "" //let it error :)
	}
}

func (s *Slim) Minify(
	ctx context.Context,
	container *Container,
	mode Optional[string],
	probeHTTP Optional[bool],
	probeExec Optional[string],
	showClogs Optional[bool],
	slimDebug Optional[bool]) (*Container, error) {
	paramMode := mode.GetOr(modeDocker)
	paramProbeHTTP := probeHTTP.GetOr(true)
	paramProbeExec := probeExec.GetOr("")
	paramShowClogs := showClogs.GetOr(false)
	paramDebug := slimDebug.GetOr(false)

	switch paramMode {
		case modeDocker, modeNative:
		default:
			paramMode = modeDocker
	}

	if paramMode != modeDocker {
		return nil, fmt.Errorf("unsupported mode - %s", paramMode)
	}

	// Start an ephemeral dockerd
	dockerd := dag.Dockerd().Service()
	// Load the input container into the dockerd
	if _, err := DockerLoad(ctx, container, dockerd); err != nil {
		if err != nil {
			return nil, err
		}
	}
	// List images on the ephemeral dockerd
	images, err := DockerImages(ctx, dockerd)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("Failed to load container into ephemeral docker engine")
	}
	firstImage := images[0]

	var cargs []string
	if paramDebug {
		cargs = append(cargs, flagDebug)
	}

	cargs = append(cargs, cmdBuild)
	cargs = append(cargs, "--tag")
	cargs = append(cargs, outputImageTag)
	cargs = append(cargs, "--target")
	cargs = append(cargs, firstImage)

	if paramShowClogs {
		cargs = append(cargs, flagShowClogs)
	}

	if paramProbeHTTP {
		cargs = append(cargs, flagHttpProbe)
	}

	if paramProbeExec != "" {
		cargs = append(cargs, flagExecProbe, paramProbeExec)
	}

	// Setup the slim container, attached to the dockerd
	slim := dag.
		Container().
		From(engineImage()).
		WithServiceBinding("dockerd", dockerd).
		WithEnvVariable("DOCKER_HOST", "tcp://dockerd:2375").
		WithExec(cargs)

	// Force execution of the slim command
	slim, err = slim.Sync(ctx)
	if err != nil {
		return container, err
	}

	// Extract the resulting image back into a container
	outputArchive := DockerClient(dockerd).WithExec([]string{
		"image", "save",
		outputImageTag,
		// firstImage, // For now we output the un-slimeed image, while we debug
		"-o", outputImageTar}).
		File(outputImageTar)
	return dag.Container().Import(outputArchive), nil
}

func DockerImages(ctx context.Context, dockerd *Service) ([]string, error) {
	raw, err := DockerClient(dockerd).
		WithExec([]string{"image", "list", "--no-trunc", "--format", "{{.ID}}"}).
		Stdout(ctx)
	if err != nil {
		return nil, err
	}
	return strings.Split(raw, "\n"), nil
}

func DockerClient(dockerd *Service) *Container {
	return dag.
		Container().
		From("index.docker.io/docker:cli").
		WithServiceBinding("dockerd", dockerd).
		WithEnvVariable("DOCKER_HOST", "tcp://dockerd:2375")
}

// Load a container into a docker engine
func DockerLoad(ctx context.Context, c *Container, dockerd *Service) (string, error) {
	client := DockerClient(dockerd).
		WithMountedFile("/tmp/container.tar", c.AsTarball())
	stdout, err := client.WithExec([]string{"load", "-i", "/tmp/container.tar"}).Stdout(ctx)
	// FIXME: parse stdout
	return stdout, err
}
