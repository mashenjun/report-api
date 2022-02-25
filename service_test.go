package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func FastTuneSample(id int64, value float64) *InsertSampleParam {
	sample := &InsertSampleParam{
		Timestamp:     time.Now().Unix(),
		Measurement:   "fast-tune-similarity",
		TiDBClusterID: "clinic",
		Fields: map[string]interface{}{
			"_value": value,
		},
		Tags: map[string]string{
			"id": fmt.Sprintf("%#x", id),
		},
	}
	return sample
}

func TestReportAPI_InsertSample(t *testing.T) {
	assert := require.New(t)
	influxdbURL := "http://localhost:8086"
	tk := "lF9VJ9pvM3xU4piExllnV800kHwGg9ie-08fTnlcZl9EcYllFMo9urMGTQ71AI3UKTJTn5D6LiRmCvDLAG9BPQ=="
	org := "my-org"
	bucket := "clinic"

	rAPI, err := NewReportAPI(influxdbURL, org, bucket, tk)
	assert.Nil(err)
	defer rAPI.Close()
	ctx := context.Background()
	{
		sampleParam := FastTuneSample(0x0001, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0100, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0101, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0102, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0103, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0104, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0200, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0201, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0202, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0203, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	{
		sampleParam := FastTuneSample(0x0204, 0.95)
		_, err = rAPI.InsertSample(ctx, sampleParam)
		assert.Nil(err)
	}
	err = rAPI.Flush(ctx)
	assert.Nil(err)
}

func TestReportAPI_QueryNodeGraph(t *testing.T) {
	assert := require.New(t)

	influxdbURL := "http://localhost:8086"
	tk := "lF9VJ9pvM3xU4piExllnV800kHwGg9ie-08fTnlcZl9EcYllFMo9urMGTQ71AI3UKTJTn5D6LiRmCvDLAG9BPQ=="
	org := "my-org"
	bucket := "clinic"

	rAPI, err := NewReportAPI(influxdbURL, org, bucket, tk)
	assert.Nil(err)
	defer rAPI.Close()
	ctx := context.Background()
	nowTS := time.Now().Unix()
	param := &QueryNodeGraphParam{
		TsRange: TsRange{
			StartTS: nowTS - 120*3600,
			EndTS:   nowTS,
		},
		TiDBClusterID: "clinic",
	}
	data, err := rAPI.QueryNodeGraph(ctx, param)
	assert.Nil(err)
	bs, err := json.Marshal(data)
	assert.Nil(err)
	t.Log(string(bs))
}
