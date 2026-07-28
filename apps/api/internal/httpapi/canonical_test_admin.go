//go:build canonicaltest

package httpapi

import (
	"net/http"
	"time"

	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/database"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/platform/objectstore"
	"github.com/MarlonJD/aviaSurveil360/apps/api/internal/testprofile"
	"github.com/go-chi/chi/v5"
)

type CanonicalTestAdmin struct {
	pool      *database.Pool
	objects   objectstore.TestResetter
	buckets   []string
	generator *testprofile.Generator
	clock     func() time.Time
}

func NewCanonicalTestAdmin(
	pool *database.Pool,
	objects objectstore.TestResetter,
	buckets []string,
	generator *testprofile.Generator,
	clock func() time.Time,
) http.Handler {
	admin := &CanonicalTestAdmin{
		pool: pool, objects: objects, buckets: buckets,
		generator: generator, clock: clock,
	}
	router := chi.NewRouter()
	router.Post("/__test/reset", admin.reset)
	return router
}

func (admin *CanonicalTestAdmin) reset(
	writer http.ResponseWriter,
	request *http.Request,
) {
	if admin.objects != nil {
		if err := admin.objects.ResetPrivateBuckets(
			request.Context(),
			admin.buckets,
		); err != nil {
			writeProblem(
				writer,
				http.StatusInternalServerError,
				"Test reset failed",
				err.Error(),
				"TEST_RESET_FAILED",
			)
			return
		}
	}
	if err := testprofile.Reset(
		request.Context(),
		admin.pool,
		admin.clock().UTC(),
	); err != nil {
		writeProblem(
			writer,
			http.StatusInternalServerError,
			"Test reset failed",
			err.Error(),
			"TEST_RESET_FAILED",
		)
		return
	}
	admin.generator.Reset()
	writeJSON(writer, http.StatusOK, map[string]string{"status": "reset"})
}
