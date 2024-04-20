package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog"
	"github.com/wolfeidau/lambda-go-extras/lambdaextras"
	lmw "github.com/wolfeidau/lambda-go-extras/middleware"
	zlog "github.com/wolfeidau/lambda-go-extras/middleware/zerolog"
)

func main() {

	ch := lmw.New(
		zlog.New(), // inject zerolog into the context
	).Then(lambdaextras.GenericHandler(processEvent))

	// use StartWithOptions as StartHandler is deprecated
	lambda.StartWithOptions(ch)
}

type FirehoseEventRecordData struct {
	TSEpochMillis int64 `json:"ts_epoch_millis,omitempty"`
}

func processEvent(ctx context.Context, evnt events.KinesisFirehoseEvent) (events.KinesisFirehoseResponse, error) {
	zerolog.Ctx(ctx).Info().
		Str("DeliveryStreamArn", evnt.DeliveryStreamArn).
		Int("RecordCount", len(evnt.Records)).
		Msg("KinesisFirehoseEvent")
	var response events.KinesisFirehoseResponse

	for _, record := range evnt.Records {

		var recordData FirehoseEventRecordData
		if err := json.Unmarshal(record.Data, &recordData); err != nil {
			return response, fmt.Errorf("failed to unmarshal record data: %w", err)
		}

		response.Records = append(response.Records, events.KinesisFirehoseResponseRecord{
			RecordID: record.RecordID,
			Result:   events.KinesisFirehoseTransformedStateOk,
			Data:     record.Data,
			Metadata: events.KinesisFirehoseResponseRecordMetadata{
				PartitionKeys: buildPartitionKeys(recordData),
			},
		})
	}

	return response, nil
}

// build the partition keys from the record data
func buildPartitionKeys(recordData FirehoseEventRecordData) map[string]string {
	ts := time.Unix(0, recordData.TSEpochMillis*int64(time.Millisecond)).UTC()
	return map[string]string{
		"year":  fmt.Sprintf("%d", ts.Year()),
		"month": fmt.Sprintf("%02d", ts.Month()),
		"day":   fmt.Sprintf("%02d", ts.Day()),
		"hour":  fmt.Sprintf("%02d", ts.Hour()),
	}
}
