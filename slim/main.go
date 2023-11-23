package main

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

const (
	//todo: multi-arch engine image
	engineImageARM = "index.docker.io/dslim/slim-arm"
	engineImageAMD = "index.docker.io/dslim/slim"
	archAMD64      = "amd64"
	archARM64      = "arm64"

	outputImageTag = "slim-output:latest"
	outputImageTar = "output.tar"

	flagDebug = "--debug"
	trueValue = "true"
	cmdBuild  = "build"

	modeDocker = "docker"
	modeNative = "native"

	flagShowClogs              = "--show-clogs"
	flagHttpProbe              = "--http-probe"
	flagHttpProbeCmd           = "--http-probe-cmd"
	flagHttpProbePorts         = "--http-probe-ports"
	flagHttpProbeExitOnFailure = "--http-probe-exit-on-failure"

	flagPublishPort         = "--publish-port"
	flagPublishExposedPorts = "--publish-exposed-ports"

	flagExecProbe = "--exec"

	flagIncludePath     = "--include-path"
	flagIncludeBin      = "--include-bin"
	flagIncludeExe      = "--include-exe"
	flagIncludeShell    = "--include-shell"
	flagIncludeNew      = "--include-new"
	flagIncludeZoneInfo = "--include-zoneinfo"
	flagPreservePath    = "--preserve-path"
	flagExcludePattern  = "--exclude-pattern"
	flagEnv             = "--env"
	flagExpose          = "--expose"
	flagContinueAfter   = "--continue-after"

	flagSensorIPCMode     = "--sensor-ipc-mode"
	flagSensorIPCEndpoint = "--sensor-ipc-endpoint"

	flagRTASourcePT      = "--rta-source-ptrace"
	flagImageBuildEngine = "--image-build-engine"
	flagImageBuildArch   = "--image-build-arch"
)

type Slim struct {
	includePaths      []string
	includeBins       []string
	includeExes       []string
	includeShell      *bool
	includeNew        *bool
	includeZoneinfo   *bool
	preservePaths     []string
	excludePatterns   []string
	envVars           []string
	sensorIPCMode     string
	sensorIPCEndpoint string
	rtaSourcePT       *bool
	imageBuildEngine  string
	imageBuildArch    string
	execProbes        []string
	httpProbeCmds     []string
	exposePorts       []string
	publishPorts      []string
}

func (s *Slim) Minify(
	ctx context.Context,
	container *Container,
	mode Optional[string],
	probeHttp Optional[bool],
	probeHttpExitOnFailure Optional[bool],
	publishExposedPorts Optional[bool],
	probeHttpPorts Optional[string], //comma separated list
	continueAfter Optional[string],
	showClogs Optional[bool],
	slimDebug Optional[bool],
) (*Container, error) {
	paramMode := mode.GetOr(modeDocker)
	paramProbeHttp := probeHttp.GetOr(true)
	paramProbeHttpPorts := probeHttpPorts.GetOr("")
	paramProbeHttpExitOnFailure := probeHttpExitOnFailure.GetOr(true)
	paramPublishExposedPorts := publishExposedPorts.GetOr(true)
	paramContinueAfter := continueAfter.GetOr("")
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

	//pick up 'false' values too
	cargs = append(cargs, flagHttpProbe, fmt.Sprintf("%v", paramProbeHttp))
	cargs = append(cargs, flagHttpProbeExitOnFailure, fmt.Sprintf("%v", paramProbeHttpExitOnFailure))
	cargs = append(cargs, flagPublishExposedPorts, fmt.Sprintf("%v", paramPublishExposedPorts))

	if paramProbeHttpPorts != "" {
		cargs = append(cargs, flagHttpProbePorts, paramProbeHttpPorts)
	}

	for _, val := range s.exposePorts {
		cargs = append(cargs, flagExpose, val)
	}

	for _, val := range s.publishPorts {
		cargs = append(cargs, flagPublishPort, val)
	}

	for _, val := range s.httpProbeCmds {
		cargs = append(cargs, flagHttpProbeCmd, val)
	}

	if len(s.execProbes) > 0 {
		//todo: support multiple exec probes (using the first one for now)
		cargs = append(cargs, flagExecProbe, s.execProbes[0])
	}

	for _, val := range s.includePaths {
		cargs = append(cargs, flagIncludePath, val)
	}

	for _, val := range s.includeBins {
		cargs = append(cargs, flagIncludeBin, val)
	}

	for _, val := range s.includeExes {
		cargs = append(cargs, flagIncludeExe, val)
	}

	for _, val := range s.preservePaths {
		cargs = append(cargs, flagPreservePath, val)
	}

	for _, val := range s.excludePatterns {
		cargs = append(cargs, flagExcludePattern, val)
	}

	for _, val := range s.envVars {
		cargs = append(cargs, flagEnv, val)
	}

	if paramContinueAfter != "" {
		cargs = append(cargs, flagContinueAfter, paramContinueAfter)
	}

	if s.sensorIPCMode != "" {
		cargs = append(cargs, flagSensorIPCMode, s.sensorIPCMode)
	}

	if s.sensorIPCEndpoint != "" {
		cargs = append(cargs, flagSensorIPCEndpoint, s.sensorIPCEndpoint)
	}

	if s.imageBuildArch != "" {
		cargs = append(cargs, flagImageBuildArch, s.imageBuildArch)
	}

	if s.imageBuildEngine != "" {
		cargs = append(cargs, flagImageBuildEngine, s.imageBuildEngine)
	}

	if s.rtaSourcePT != nil {
		cargs = append(cargs, flagRTASourcePT, fmt.Sprintf("%v", *s.rtaSourcePT))
	}

	if s.includeZoneinfo != nil {
		cargs = append(cargs, flagIncludeZoneInfo, fmt.Sprintf("%v", *s.includeZoneinfo))
	}

	if s.includeNew != nil {
		cargs = append(cargs, flagIncludeNew, fmt.Sprintf("%v", *s.includeNew))
	}

	if s.includeShell != nil {
		cargs = append(cargs, flagIncludeShell, fmt.Sprintf("%v", *s.includeShell))
	}

	//reuse the param to show the constructed command line:
	if paramDebug {
		fmt.Printf("Slim(Toolkit) params: %#v\n", cargs)
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

func (s *Slim) Compare(
	ctx context.Context,
	container *Container,
	showClogs Optional[bool],
	slimDebug Optional[bool],
) (*Container, error) {
	slimmed, err := s.Minify(ctx,
		container,
		OptEmpty[string](), //mode
		OptEmpty[bool](),   //probeHTTP
		OptEmpty[bool](),   //probeHTTPExitOnFailure
		OptEmpty[bool](),   //publishExposedPorts
		OptEmpty[string](), //probeHTTPPorts
		OptEmpty[string](), //continueAfter
		showClogs,
		slimDebug)
	if err != nil {
		return nil, err
	}

	debug := dag.
		Container().
		From("alpine").
		WithMountedDirectory("/image.before", slimmed.Rootfs()).
		WithMountedDirectory("/image.after", container.Rootfs())
	return debug, nil
}

// MORE OPTIONAL PARAMS

func (s *Slim) WithIncludePath(val string) *Slim {
	s.includePaths = append(s.includePaths, val)
	return s
}

func (s *Slim) WithIncludeBin(val string) *Slim {
	s.includeBins = append(s.includeBins, val)
	return s
}

func (s *Slim) WithIncludeExe(val string) *Slim {
	s.includeExes = append(s.includeExes, val)
	return s
}

func (s *Slim) WithIncludeShell(val bool) *Slim {
	s.includeShell = &val
	return s
}

func (s *Slim) WithIncludeNew(val bool) *Slim {
	s.includeNew = &val
	return s
}

func (s *Slim) WithIncludeZoneinfo(val bool) *Slim {
	s.includeZoneinfo = &val
	return s
}

func (s *Slim) WithPreservePath(val string) *Slim {
	s.preservePaths = append(s.preservePaths, val)
	return s
}

func (s *Slim) WithExcludePattern(val string) *Slim {
	s.excludePatterns = append(s.excludePatterns, val)
	return s
}

func (s *Slim) WithEnv(val string) *Slim {
	s.envVars = append(s.envVars, val)
	return s
}

func (s *Slim) WithSensorIpcMode(val string) *Slim {
	s.sensorIPCMode = val
	return s
}

func (s *Slim) WithSensorIpcEndpoint(val string) *Slim {
	s.sensorIPCEndpoint = val
	return s
}

func (s *Slim) WithSourcePtrace(val bool) *Slim {
	s.rtaSourcePT = &val
	return s
}

func (s *Slim) WithImageBuildEngine(val string) *Slim {
	s.imageBuildEngine = val
	return s
}

func (s *Slim) WithImageBuildArch(val string) *Slim {
	s.imageBuildArch = val
	return s
}

func (s *Slim) WithExecProbe(val string) *Slim {
	s.execProbes = append(s.execProbes, val)
	return s
}

func (s *Slim) WithHttpProbeCmd(val string) *Slim {
	s.httpProbeCmds = append(s.httpProbeCmds, val)
	return s
}

func (s *Slim) WithExposePort(val string) *Slim {
	s.exposePorts = append(s.exposePorts, val)
	return s
}

func (s *Slim) WithPublishPort(val string) *Slim {
	s.publishPorts = append(s.publishPorts, val)
	return s
}

// SUPPORTING FUNCTIONS:

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
