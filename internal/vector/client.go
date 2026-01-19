package vector

import (
	"context"
	"fmt"

	pb "github.com/qdrant/go-client/qdrant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.QdrantClient
	points pb.PointsClient
	colls  pb.CollectionsClient
}

func NewClient(addr string) (*Client, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewQdrantClient(conn),
		points: pb.NewPointsClient(conn),
		colls:  pb.NewCollectionsClient(conn),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) CreateCollection(ctx context.Context, name string, vectorSize uint64) error {
	exists, err := c.CollectionExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = c.colls.Create(ctx, &pb.CreateCollection{
		CollectionName: name,
		VectorsConfig: &pb.VectorsConfig{
			Config: &pb.VectorsConfig_Params{
				Params: &pb.VectorParams{
					Size:     vectorSize,
					Distance: pb.Distance_Cosine,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", name, err)
	}
	return nil
}

func (c *Client) CollectionExists(ctx context.Context, name string) (bool, error) {
	resp, err := c.colls.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return false, fmt.Errorf("failed to list collections: %w", err)
	}

	for _, col := range resp.Collections {
		if col.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) ListCollections(ctx context.Context) ([]string, error) {
	resp, err := c.colls.List(ctx, &pb.ListCollectionsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	var names []string
	for _, col := range resp.Collections {
		names = append(names, col.Name)
	}
	return names, nil
}

func (c *Client) UpsertPoints(ctx context.Context, collectionName string, points []*pb.PointStruct) error {
	// Simple batch upsert
	_, err := c.points.Upsert(ctx, &pb.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
		Wait:           ptr(true),
	})
	return err
}

func (c *Client) Search(ctx context.Context, collectionName string, vector []float32, limit uint64) ([]*pb.ScoredPoint, error) {
	resp, err := c.points.Search(ctx, &pb.SearchPoints{
		CollectionName: collectionName,
		Vector:         vector,
		Limit:          limit,
		WithPayload: &pb.WithPayloadSelector{
			SelectorOptions: &pb.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search collection %s: %w", collectionName, err)
	}
	return resp.Result, nil
}

func ptr(b bool) *bool {
	return &b
}
