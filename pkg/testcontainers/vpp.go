package testcontainers

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types/container"
	"github.com/testcontainers/testcontainers-go"
)

const (
	VPPVersion  = "24.02"
	VPPImage    = "ligato/vpp-base:" + VPPVersion
	ReaperImage = "0.7.0"
)

type vppContainer struct {
	vppContainer testcontainers.Container
}

func CreateVppContainer(ctx context.Context, vppExposePort string) (*vppContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        VPPImage,
		ReaperImage:  ReaperImage,
		ExposedPorts: []string{fmt.Sprintf("%s/tcp", vppExposePort)},
		HostConfigModifier: func(hostConfig *container.HostConfig) {
			hostConfig.AutoRemove = false
		},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "vpp_startup.conf",
				ContainerFilePath: "/etc/vpp/startup.conf",
				FileMode:          777,
			},
		},
		CapAdd:     []string{"IPC_LOCK", "NET_ADMIN", "SYS_NICE"},
		Privileged: true,
	}

	vppCont, err := testcontainers.GenericContainer(
		ctx,
		testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		},
	)
	if err != nil {
		return nil, err
	}

	if err = vppCont.StartLogProducer(ctx); err != nil {
		log.Fatal(err)
	}

	lc := LogConsumer{}

	vppCont.FollowOutput(&lc)

	return &vppContainer{vppContainer: vppCont}, nil
}

func (t *vppContainer) DeleteVppContainer(ctx context.Context) {
	_ = t.vppContainer.StopLogProducer()
	_ = t.vppContainer.Terminate(ctx)
}

func (t *vppContainer) GetIP(ctx context.Context) string {
	addr, _ := t.vppContainer.ContainerIP(ctx)

	return addr
}

func (t *vppContainer) GetHost(ctx context.Context) string {
	host, _ := t.vppContainer.Host(ctx)

	return host
}

func (t *vppContainer) ExecCmd(ctx context.Context, commands []string) {
	_, _, _ = t.vppContainer.Exec(ctx, commands)
}

type LogConsumer struct{}

func (g *LogConsumer) Accept(l testcontainers.Log) {
	fmt.Println(string(l.Content))
}
