package mongodb

import (
	"context"
	"errors"
	"log"
	"time"

	"orchestrator/internal/domain/entity"
	"orchestrator/internal/domain/repository"
	"orchestrator/internal/infrastructure/metrics"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoJobRepo struct {
	jobsCol *mongo.Collection
}

func NewMongoJobRepo(db *mongo.Database) repository.JobRepository {
	col := db.Collection("jobs")

	_, _ = col.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys: bson.D{bson.E{Key: "status", Value: 1}},
	})

	return &MongoJobRepo{
		jobsCol: col,
	}
}

func (r *MongoJobRepo) Create(ctx context.Context, job *entity.Job) error {
	metrics.IncJobsCreated()

	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	_, err := r.jobsCol.InsertOne(ctx, job)
	if err != nil {
		metrics.IncError("mongo_job_repo", "create_error")
		return err
	}
	return nil
}

func (r *MongoJobRepo) GetByID(ctx context.Context, id string) (*entity.Job, error) {
	metrics.IncDBFileOp("get")

	var job entity.Job
	err := r.jobsCol.FindOne(ctx, bson.M{"id": id}).Decode(&job)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		metrics.IncError("mongo_job_repo", "get_error")
		return nil, err
	}
	return &job, nil
}

func (r *MongoJobRepo) List(ctx context.Context) ([]*entity.Job, error) {
	metrics.IncDBFileOp("list")

	cur, err := r.jobsCol.Find(ctx, bson.D{})
	if err != nil {
		metrics.IncError("mongo_job_repo", "list_error")
		return nil, err
	}
	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			log.Printf("close body err: %s", err)
		}
	}()

	var jobs []*entity.Job
	for cur.Next(ctx) {
		var j entity.Job
		if err := cur.Decode(&j); err != nil {
			metrics.IncError("mongo_job_repo", "list_decode_error")
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	if err := cur.Err(); err != nil {
		metrics.IncError("mongo_job_repo", "list_cursor_error")
	}
	return jobs, cur.Err()
}

func (r *MongoJobRepo) ListByStatus(ctx context.Context, status entity.JobStatus) ([]*entity.Job, error) {
	metrics.IncDBFileOp("list")

	cur, err := r.jobsCol.Find(ctx, bson.M{"status": status})
	if err != nil {
		metrics.IncError("mongo_job_repo", "list_by_status_error")
		return nil, err
	}
	defer func() {
		err := cur.Close(ctx)
		if err != nil {
			log.Printf("close body err: %s", err)
		}
	}()

	var jobs []*entity.Job
	for cur.Next(ctx) {
		var j entity.Job
		if err := cur.Decode(&j); err != nil {
			metrics.IncError("mongo_job_repo", "list_by_status_decode_error")
			return nil, err
		}
		jobs = append(jobs, &j)
	}
	if err := cur.Err(); err != nil {
		metrics.IncError("mongo_job_repo", "list_by_status_cursor_error")
	}
	return jobs, cur.Err()
}

func (r *MongoJobRepo) Update(ctx context.Context, job *entity.Job) error {
	metrics.IncDBFileOp("put")

	job.UpdatedAt = time.Now()
	res, err := r.jobsCol.ReplaceOne(ctx, bson.M{"id": job.ID}, job)
	if err != nil {
		metrics.IncError("mongo_job_repo", "update_error")
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MongoJobRepo) UpdateStatus(ctx context.Context, id string, status entity.JobStatus) error {
	metrics.IncDBFileOp("put")

	filter := bson.M{"id": id}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	res, err := r.jobsCol.UpdateOne(ctx, filter, update)
	if err != nil {
		metrics.IncError("mongo_job_repo", "update_status_error")
		return err
	}
	if res.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MongoJobRepo) Delete(ctx context.Context, id string) error {
	metrics.IncDBFileOp("delete")

	res, err := r.jobsCol.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		metrics.IncError("mongo_job_repo", "delete_error")
		return err
	}
	if res.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (r *MongoJobRepo) CountByStatus(ctx context.Context, status entity.JobStatus) (int, error) {
	metrics.IncDBFileOp("count")

	count, err := r.jobsCol.CountDocuments(ctx, bson.M{"status": status})
	if err != nil {
		metrics.IncError("mongo_job_repo", "count_by_status_error")
		return 0, err
	}
	return int(count), nil
}
