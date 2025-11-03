package mongodb

import (
	"context"
	"log"

	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
	"orchestrator/internal/infrastructure/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoConfigRepo struct {
	col *mongo.Collection
}

func NewMongoConfigRepo(db *mongo.Database) repository.ConfgiFileRepository {
	col := db.Collection("config_files")

	_, _ = col.Indexes().CreateMany(context.Background(), []mongo.IndexModel{
		{Keys: bson.D{bson.E{Key: "jobid", Value: 1}}},
	})

	return &MongoConfigRepo{
		col: col,
	}
}

func (r *MongoConfigRepo) SaveFiles(ctx context.Context, files []*entity.ConfigFile) error {
	if len(files) == 0 {
		return nil
	}

	metrics.IncDBFileOp("put")

	docs := make([]interface{}, len(files))
	for i, f := range files {
		docs[i] = f
	}

	_, err := r.col.InsertMany(ctx, docs)
	if err != nil {
		metrics.IncError("mongo_config_repo", "save_error")
		return err
	}
	return nil
}

func (r *MongoConfigRepo) GetFiles(ctx context.Context, job_id string) ([]*entity.ConfigFile, error) {
	metrics.IncDBFileOp("get")

	filter := bson.M{"jobid": job_id}
	files, err := r.findFiles(ctx, filter)
	if err != nil {
		metrics.IncError("mongo_config_repo", "get_error")
		return nil, err
	}
	return files, nil
}

func (r *MongoConfigRepo) GetFilesByJobID(ctx context.Context, jobID string) ([]*entity.ConfigFile, error) {
	metrics.IncDBFileOp("get")

	filter := bson.M{"jobid": jobID}
	files, err := r.findFiles(ctx, filter)
	if err != nil {
		metrics.IncError("mongo_config_repo", "get_by_jobid_error")
		return nil, err
	}
	return files, nil
}

func (r *MongoConfigRepo) ListRequests(ctx context.Context) ([]string, error) {
	metrics.IncDBFileOp("list")

	values, err := r.col.Distinct(ctx, "request_id", bson.D{})
	if err != nil {
		metrics.IncError("mongo_config_repo", "list_error")
		return nil, err
	}

	requests := make([]string, 0, len(values))
	for _, v := range values {
		if s, ok := v.(string); ok {
			requests = append(requests, s)
		}
	}
	return requests, nil
}

func (r *MongoConfigRepo) DeleteRequest(ctx context.Context, requestID string) error {
	metrics.IncDBFileOp("delete")

	_, err := r.col.DeleteMany(ctx, bson.M{"job_id": requestID})
	if err != nil {
		metrics.IncError("mongo_config_repo", "delete_error")
		return err
	}
	return nil
}

func (r *MongoConfigRepo) findFiles(ctx context.Context, filter bson.M) ([]*entity.ConfigFile, error) {
	cur, err := r.col.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			log.Printf("close body err: %s", err)
		}
	}()

	var result []*entity.ConfigFile
	for cur.Next(ctx) {
		var doc entity.ConfigFile
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		result = append(result, &doc)
	}
	return result, cur.Err()
}
