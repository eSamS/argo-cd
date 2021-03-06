package apiclient

import (
	"crypto/tls"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/argoproj/argo-cd/util"
	argogrpc "github.com/argoproj/argo-cd/util/grpc"
)

// Clientset represets repository server api clients
type Clientset interface {
	NewRepoServerClient() (util.Closer, RepoServerServiceClient, error)
}

type clientSet struct {
	address        string
	timeoutSeconds int
}

func (c *clientSet) NewRepoServerClient() (util.Closer, RepoServerServiceClient, error) {
	retryOpts := []grpc_retry.CallOption{
		grpc_retry.WithMax(3),
		grpc_retry.WithBackoff(grpc_retry.BackoffLinear(1000 * time.Millisecond)),
	}
	unaryInterceptors := []grpc.UnaryClientInterceptor{grpc_retry.UnaryClientInterceptor(retryOpts...)}
	if c.timeoutSeconds > 0 {
		unaryInterceptors = append(unaryInterceptors, argogrpc.WithTimeout(time.Duration(c.timeoutSeconds)*time.Second))
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})),
		grpc.WithStreamInterceptor(grpc_retry.StreamClientInterceptor(retryOpts...)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryInterceptors...)),
	}

	conn, err := grpc.Dial(c.address, opts...)
	if err != nil {
		log.Errorf("Unable to connect to repository service with address %s", c.address)
		return nil, nil, err
	}
	return conn, NewRepoServerServiceClient(conn), nil
}

// NewRepoServerClientset creates new instance of repo server Clientset
func NewRepoServerClientset(address string, timeoutSeconds int) Clientset {
	return &clientSet{address: address, timeoutSeconds: timeoutSeconds}
}
