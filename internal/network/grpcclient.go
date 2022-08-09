package network

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	pb "github.com/ITA-Dnipro/Dp-230-Result-Collector/proto"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	"parabellum.crawler/internal/crawler"

	"google.golang.org/grpc"
)

const err5xxType = "5XX-error"

type ClientGRPC struct {
	client     pb.ReportServiceClient
	connection *grpc.ClientConn
}

func NewClient(serverAddr string) *ClientGRPC {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Panicln("grpc connection failed\t", err)
	}

	return &ClientGRPC{
		client:     pb.NewReportServiceClient(conn),
		connection: conn,
	}
}

func (cl *ClientGRPC) Push5XXResult(ctx context.Context, taskID string, fitResponses []*crawler.Response) error {
	results := make([]*pb.Result, 0, len(fitResponses))
	for _, resp := range fitResponses {
		results = append(results, &pb.Result{
			URL:       resp.VisitedLink.URL,
			StartTime: timestamppb.New(time.Now()),
			PoCs: []*pb.PoC{
				{
					Type:     err5xxType,
					Payload:  fmt.Sprintf("HTTP status:\t%d %s", resp.StatusCode, http.StatusText(resp.StatusCode)),
					Data:     resp.VisitedLink.URL,
					Evidence: resp.VisitedLink.URL,
				},
			},
		})
	}

	req := &pb.PushResultReq{
		ID: taskID,
		TestResult: &pb.TestResult{
			Type:    err5xxType,
			Results: results,
		},
	}
	_, err := cl.client.PushResult(ctx, req)

	return err
}

func (cl *ClientGRPC) Close() error {
	return cl.connection.Close()
}
