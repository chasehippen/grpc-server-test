package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "io/ioutil"
    "log"
    "net"
    "os"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/health/grpc_health_v1"
    pb "grpc-server/greeter"
)

type server struct {
    pb.UnimplementedGreeterServer
    grpc_health_v1.UnimplementedHealthServer
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
    log.Printf("Received: %v", in.GetName())
    return &pb.HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *server) Check(ctx context.Context, in *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
    return &grpc_health_v1.HealthCheckResponse{
        Status: grpc_health_v1.HealthCheckResponse_SERVING,
    }, nil
}

func main() {
    log.Println("Starting server...")
    lis, err := net.Listen("tcp", ":9091")
    if err != nil {
        log.Fatalf("failed to listen: %v", err)
    }
    log.Println("Listening on :9091...")

    var opts []grpc.ServerOption

    if os.Getenv("SERVER_TLS_ENABLED") == "true" {
        log.Println("TLS is enabled")
        certPath := os.Getenv("TLS_CERT_PATH")
        if certPath == "" {
            certPath = "/etc/tls"
        }
        log.Printf("Using certificate path: %s", certPath)

        cert, err := tls.LoadX509KeyPair(certPath+"/tls.crt", certPath+"/tls.key")
        if err != nil {
            log.Fatalf("failed to load key pair: %s", err)
        }
        log.Println("Loaded key pair")

        ca, err := ioutil.ReadFile(certPath + "/ca.crt")
        if err != nil {
            log.Fatalf("could not read ca certificate: %s", err)
        }
        log.Println("Read CA certificate")

        caPool := x509.NewCertPool()
        if ok := caPool.AppendCertsFromPEM(ca); !ok {
            log.Fatalf("failed to append ca certs")
        }
        log.Println("Appended CA certs to pool")

        creds := credentials.NewTLS(&tls.Config{
            Certificates: []tls.Certificate{cert},
            ClientCAs:    caPool,
	    RootCAs:      caPool,
            ClientAuth:   tls.RequireAndVerifyClientCert,
        })
        log.Println("Created TLS credentials")

        opts = append(opts, grpc.Creds(creds))
    }

    s := grpc.NewServer(opts...)

    pb.RegisterGreeterServer(s, &server{})
    grpc_health_v1.RegisterHealthServer(s, &server{})

    log.Println("Greeter service and health check service registered...")
    if err := s.Serve(lis); err != nil {
        log.Fatalf("failed to serve: %v", err)
    }
    log.Println("Server stopped.")
}
