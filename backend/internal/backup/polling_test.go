package backup

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegoapi"
)

func TestPollingPreservesOffsetAcrossFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls, reconnected := 0, 0
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		started := time.Now()
		caller := callerFunc(func(_ context.Context, _ string, data *telegoapi.RequestData) (*telegoapi.Response, error) {
			var params telego.GetUpdatesParams
			if err := json.Unmarshal(data.BodyRaw, &params); err != nil {
				t.Errorf("decode request: %v", err)
				cancel()
				return nil, err
			}
			wantOffsets := []int{0, 42, 42, 43, 43, 43, 43}
			if calls >= len(wantOffsets) {
				t.Error("unexpected extra poll")
				cancel()
				return nil, context.Canceled
			}
			if params.Offset != wantOffsets[calls] || params.Timeout != 30 || !reflect.DeepEqual(params.AllowedUpdates, []string{"message"}) {
				t.Errorf("poll %d: unexpected params %+v", calls, params)
			}
			calls++
			switch calls {
			case 1:
				return &telegoapi.Response{Ok: true, Result: []byte(`[{"update_id":41}]`)}, nil
			case 2:
				return nil, errors.New("dial tcp: i/o timeout")
			case 3:
				return &telegoapi.Response{Ok: true, Result: []byte(`[{"update_id":41},{"update_id":42}]`)}, nil
			case 4:
				return &telegoapi.Response{Error: &telegoapi.Error{ErrorCode: 502}}, nil
			case 5:
				return &telegoapi.Response{Ok: true, Result: []byte(`"invalid updates"`)}, nil
			case 6:
				return &telegoapi.Response{Ok: true, Result: []byte(`[]`)}, nil
			default:
				cancel()
				return nil, context.Canceled
			}
		})
		bot, err := telego.NewBot("1234567890:aaaabbbbaaaabbbbaaaabbbbaaaabbbbccc", telego.WithAPICaller(caller))
		if err != nil {
			t.Fatal(err)
		}
		var received []int
		for update := range UpdatesViaLongPolling(ctx, bot, func(context.Context) { reconnected++ }, logger) {
			received = append(received, update.UpdateID)
		}
		if !reflect.DeepEqual(received, []int{41, 42}) || calls != 7 || reconnected != 3 {
			t.Fatalf("updates=%v calls=%d reconnects=%d", received, calls, reconnected)
		}
		if elapsed := time.Since(started); elapsed != 6*time.Second {
			t.Fatalf("retry delay total = %s, want 6s", elapsed)
		}
	})
}

func TestPollingTimeoutReconnectsAndCancellationStopsRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls, reconnected := 0, 0
		logger := slog.New(slog.NewTextHandler(io.Discard, nil))
		bot, err := telego.NewBot("1234567890:aaaabbbbaaaabbbbaaaabbbbaaaabbbbccc", telego.WithAPICaller(callerFunc(
			func(ctx context.Context, _ string, _ *telegoapi.RequestData) (*telegoapi.Response, error) {
				calls++
				<-ctx.Done()
				return nil, ctx.Err()
			})))
		if err != nil {
			t.Fatal(err)
		}
		started := time.Now()
		updates := UpdatesViaLongPolling(ctx, bot, func(context.Context) {
			reconnected++
			cancel()
		}, logger)
		for range updates {
			t.Error("unexpected update")
		}
		if calls != 1 || reconnected != 1 || time.Since(started) != 35*time.Second {
			t.Fatalf("calls=%d reconnects=%d elapsed=%s", calls, reconnected, time.Since(started))
		}
	})
}
