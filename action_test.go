package stream

import (
	"encoding/json"
	"testing"

	"github.com/qingcloudhx/core/action"
	"github.com/qingcloudhx/core/app/resource"
	"github.com/qingcloudhx/core/engine/channels"
	"github.com/qingcloudhx/core/support/test"
	"github.com/qingcloudhx/stream/pipeline"
	"github.com/stretchr/testify/assert"
)

const testConfig string = `{
  "id": "flogo-stream",
  "ref": "github.com/qingcloudhx/stream",
  "settings": {
    "pipelineURI": "res://pipeline:test",
    "outputChannel": "testChan"
  }
}
`
const resData string = `{
        "metadata": {
          "input": [
            {
              "name": "input",
              "type": "integer"
            }
          ]
        },
        "stages": [
        ]
      }`

func TestActionFactory_New(t *testing.T) {

	cfg := &action.Config{}
	err := json.Unmarshal([]byte(testConfig), cfg)

	if err != nil {
		t.Error(err)
		return
	}

	af := ActionFactory{}
	ctx := test.NewActionInitCtx()

	err = af.Initialize(ctx)
	assert.Nil(t, err)

	resourceCfg := &resource.Config{ID: "pipeline:test"}
	resourceCfg.Data = []byte(resData)
	err = ctx.AddResource(pipeline.ResType, resourceCfg)
	assert.Nil(t, err)

	_, err = channels.New("testChan", 5)
	assert.Nil(t, err)

	act, err := af.New(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, act)
}
