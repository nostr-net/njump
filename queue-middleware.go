package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/puzpuzpuz/xsync/v3"
	"github.com/segmentio/fasthash/fnv1a"

	"golang.org/x/sync/semaphore"
)

type queuedReq struct {
	w http.ResponseWriter
	r *http.Request
}

var buckets []*semaphore.Weighted

var (
	queueAcquireTimeoutError          = errors.New("QAT")
	redirectToCloudflareCacheHitMaybe = errors.New("RTCCHM")
	requestCanceledAbortEverything    = errors.New("RCAE")
	serverUnderHeavyLoad              = errors.New("SUHL")
)

var inCourse = xsync.NewMapOfWithHasher[uint64, struct{}](
	func(key uint64, seed uint64) uint64 { return key },
)

func initQueueBuckets(size int) {
	if size < 1 {
		size = 1
	}
	buckets = make([]*semaphore.Weighted, 52)
	for i := range buckets {
		buckets[i] = semaphore.NewWeighted(int64(size))
	}
}

func await(ctx context.Context) {
	val := ctx.Value("ticket")
	if val == nil {
		return
	}
	req, _ := ctx.Value("request").(*http.Request)
	code := val.(int)

	reqNum := ctx.Value("reqNum").(uint64)
	if _, ok := inCourse.LoadOrStore(reqNum, struct{}{}); ok {
		// we've already acquired a semaphore for this request, no need to do it again
		return
	}

	sem := buckets[code]
	if sem.TryAcquire(1) {
		// means we're the first to use this bucket
		recordQueueOutcome(req, "acquired")
		go func() {
			// we'll release it after the request is answered
			<-ctx.Done()
			sem.Release(1)
		}()
	} else {
		// otherwise someone else has already locked it, so we wait
		acquireTimeout, cancel := context.WithTimeoutCause(ctx, queueAcquireTimeoutDuration(), queueAcquireTimeoutError)
		defer cancel()

		err := sem.Acquire(acquireTimeout, 1)
		if err == nil {
			// got it soon enough
			sem.Release(1)
			recordQueueOutcome(req, "redirect")
			panic(redirectToCloudflareCacheHitMaybe)
		} else if context.Cause(acquireTimeout) == queueAcquireTimeoutError {
			// took too long
			recordQueueOutcome(req, "overload")
			panic(serverUnderHeavyLoad)
		} else {
			// request was canceled
			recordQueueOutcome(req, "canceled")
			panic(requestCanceledAbortEverything)
		}
	}
}

var reqNumSource atomic.Uint64

func queueMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if isMetricsPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/favicon.ico" || strings.HasPrefix(r.URL.Path, "/njump/static/") {
			next.ServeHTTP(w, r)
			return
		}

		reqNum := reqNumSource.Add(1)

		// these will be used when we later call await(ctx)
		ticket := int(fnv1a.HashString64(r.URL.Path) % uint64(len(buckets)))
		ctx := context.WithValue(
			context.WithValue(
				context.WithValue(
					r.Context(),
					"reqNum", reqNum,
				),
				"request", r,
			),
			"ticket", ticket,
		)

		defer func() {
			err := recover()

			if err == nil {
				return
			}

			switch err {
			// if we are not the first to request this we will wait for the underlying page to be loaded
			// then we will be redirect to open it again, so hopefully we will hit the cloudflare cache this time
			case redirectToCloudflareCacheHitMaybe:
				path := r.URL.Path
				if r.URL.RawQuery != "" {
					path += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, path, http.StatusFound)

			case serverUnderHeavyLoad:
				w.WriteHeader(504)
				w.Write([]byte("server under heavy load, please try again in a couple of seconds"))
				return

			case requestCanceledAbortEverything:
				return

			default:
				recordPanic(r, "queue")
				trace := trackError(r, err)
				log.Error().Any("panic", err).Str("path", r.URL.Path).Msg("panic recovered in queue middleware")
				writeInternalServerError(w, r, trace)
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))

		// cleanup this
		inCourse.Delete(reqNum)
	}
}

func queueAcquireTimeoutDuration() time.Duration {
	timeout := time.Duration(s.QueueAcquireTimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 6 * time.Second
	}
	return timeout
}
