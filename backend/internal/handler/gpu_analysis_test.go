// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
)

func TestConfirmAndStopJobSucceedsWhenJobWasAlreadyDeleted(t *testing.T) {
	db := newGpuAnalysisHandlerTestDB(t, "confirm_stop_deleted_job")
	const jobID uint = 1
	const jobName = "deleted-job"
	if err := db.Exec(
		"INSERT INTO jobs (id, job_name, status, deleted_at) VALUES (?, ?, ?, ?)",
		jobID,
		jobName,
		batch.Completed,
		time.Now(),
	).Error; err != nil {
		t.Fatal(err)
	}

	analysis := model.GpuAnalysis{
		JobID:        jobID,
		JobName:      jobName,
		ReviewStatus: model.ReviewStatusPending,
	}
	if err := db.Create(&analysis).Error; err != nil {
		t.Fatal(err)
	}

	mgr := &GpuAnalysisMgr{
		client: newGpuAnalysisFakeClient(t),
	}
	recorder := requestConfirmAndStopJob(t, mgr, analysis.ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("POST confirm-stop returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}

	var updatedAnalysis model.GpuAnalysis
	if err := db.First(&updatedAnalysis, analysis.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updatedAnalysis.ReviewStatus != model.ReviewStatusConfirmed {
		t.Fatalf("review status = %d, want %d", updatedAnalysis.ReviewStatus, model.ReviewStatusConfirmed)
	}

	var deletedJob model.Job
	if err := db.Unscoped().First(&deletedJob, jobID).Error; err != nil {
		t.Fatal(err)
	}
	if !deletedJob.DeletedAt.Valid {
		t.Fatal("confirm-stop restored or modified the soft-deleted job record")
	}
	if deletedJob.Status != batch.Completed {
		t.Fatalf("deleted job status = %s, want %s", deletedJob.Status, batch.Completed)
	}
}

func TestConfirmAndStopJobDoesNotMarkReviewWhenKubernetesLookupFails(t *testing.T) {
	db := newGpuAnalysisHandlerTestDB(t, "confirm_stop_kubernetes_error")
	analysis := model.GpuAnalysis{
		JobName:      "unavailable-job",
		ReviewStatus: model.ReviewStatusPending,
	}
	if err := db.Create(&analysis).Error; err != nil {
		t.Fatal(err)
	}

	mgr := &GpuAnalysisMgr{
		client: failingGetClient{
			Client: newGpuAnalysisFakeClient(t),
			err:    io.ErrUnexpectedEOF,
		},
	}
	recorder := requestConfirmAndStopJob(t, mgr, analysis.ID)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("POST confirm-stop returned HTTP %d: %s", recorder.Code, recorder.Body.String())
	}

	var unchangedAnalysis model.GpuAnalysis
	if err := db.First(&unchangedAnalysis, analysis.ID).Error; err != nil {
		t.Fatal(err)
	}
	if unchangedAnalysis.ReviewStatus != model.ReviewStatusPending {
		t.Fatalf("review status = %d, want pending after stop failure", unchangedAnalysis.ReviewStatus)
	}
}

func newGpuAnalysisHandlerTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.GpuAnalysis{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE jobs (
		id INTEGER PRIMARY KEY,
		deleted_at DATETIME,
		job_name TEXT NOT NULL,
		status TEXT,
		completed_timestamp DATETIME
	)`).Error; err != nil {
		t.Fatal(err)
	}
	query.SetDefault(db)
	return db
}

func newGpuAnalysisFakeClient(t *testing.T) client.Client {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := batch.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return fake.NewClientBuilder().WithScheme(scheme).Build()
}

func requestConfirmAndStopJob(t *testing.T, mgr *GpuAnalysisMgr, analysisID uint) *httptest.ResponseRecorder {
	t.Helper()
	router := gin.New()
	mgr.RegisterAdmin(router.Group("/v1/admin/gpu-analysis"))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/gpu-analysis/"+fmt.Sprint(analysisID)+"/confirm-stop",
		http.NoBody,
	)
	router.ServeHTTP(recorder, request)
	return recorder
}

type failingGetClient struct {
	client.Client
	err error
}

func (c failingGetClient) Get(context.Context, client.ObjectKey, client.Object, ...client.GetOption) error {
	return c.err
}
