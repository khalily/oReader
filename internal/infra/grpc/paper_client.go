package grpc

import (
	"context"
	"io"
	"time"

	pb "oreader/converter/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// ConvertProgress represents a progress update from the converter
type ConvertProgress struct {
	Status   string
	Progress int32
	Markdown string
	Error    string
}

// PaperMetadata represents extracted paper metadata
type PaperMetadata struct {
	Title         string
	Authors       []string
	Abstract      string
	Keywords      []string
	PublishedYear string
	DOI           string
}

// PaperConverterClient defines the interface for the paper converter gRPC client
type PaperConverterClient interface {
	Convert(ctx context.Context, pdfContent []byte, filename string) ([]ConvertProgress, error)
	ExtractMetadata(ctx context.Context, markdown string) (*PaperMetadata, error)
	Close() error
}

type paperClient struct {
	conn   *grpc.ClientConn
	client pb.PaperConverterClient
}

// NewPaperClient creates a new gRPC client for the paper converter service
func NewPaperClient(addr string, timeout time.Duration) (PaperConverterClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	return &paperClient{
		conn:   conn,
		client: pb.NewPaperConverterClient(conn),
	}, nil
}

func (c *paperClient) Convert(ctx context.Context, pdfContent []byte, filename string) ([]ConvertProgress, error) {
	stream, err := c.client.Convert(ctx, &pb.PdfConvertRequest{
		PdfContent: pdfContent,
		Filename:   filename,
	})
	if err != nil {
		return nil, err
	}

	var results []ConvertProgress
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return results, err
		}
		results = append(results, ConvertProgress{
			Status:   resp.Status,
			Progress: resp.Progress,
			Markdown: resp.Markdown,
			Error:    resp.Error,
		})
	}
	return results, nil
}

func (c *paperClient) ExtractMetadata(ctx context.Context, markdown string) (*PaperMetadata, error) {
	resp, err := c.client.ExtractMetadata(ctx, &pb.MetadataRequest{
		Markdown: markdown,
	})
	if err != nil {
		return nil, err
	}

	return &PaperMetadata{
		Title:         resp.Title,
		Authors:       resp.Authors,
		Abstract:      resp.Abstract,
		Keywords:      resp.Keywords,
		PublishedYear: resp.PublishedYear,
		DOI:           resp.Doi,
	}, nil
}

func (c *paperClient) Close() error {
	return c.conn.Close()
}
