package stream

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qingcloudhx/core/action"
	"github.com/qingcloudhx/core/app/resource"
	"github.com/qingcloudhx/core/data/coerce"
	"github.com/qingcloudhx/core/data/mapper"
	"github.com/qingcloudhx/core/data/metadata"
	"github.com/qingcloudhx/core/engine/channels"
	"github.com/qingcloudhx/core/support/log"
	"github.com/qingcloudhx/stream/pipeline"
)

func init() {
	_ = action.Register(&StreamAction{}, &ActionFactory{})
}

var manager *pipeline.Manager
var actionMd = action.ToMetadata(&Settings{})
var logger log.Logger

type Settings struct {
	PipelineURI   string `md:"pipelineURI,required"`
	GroupBy       string `md:"groupBy"`
	OutputChannel string `md:"outputChannel"`
}

type ActionFactory struct {
	resManager *resource.Manager
}

func (f *ActionFactory) Initialize(ctx action.InitContext) error {

	f.resManager = ctx.ResourceManager()

	logger = log.ChildLogger(log.RootLogger(), "pipeline")

	if manager != nil {
		return nil
	}

	mapperFactory := mapper.NewFactory(pipeline.GetDataResolver())

	manager = pipeline.NewManager()
	err := resource.RegisterLoader(pipeline.ResType, pipeline.NewResourceLoader(mapperFactory, pipeline.GetDataResolver()))
	return err
}

func (f *ActionFactory) New(config *action.Config) (action.Action, error) {

	settings := &Settings{}
	err := metadata.MapToStruct(config.Settings, settings, true)
	if err != nil {
		return nil, err
	}

	streamAction := &StreamAction{}

	if settings.PipelineURI == "" {
		return nil, fmt.Errorf("pipeline URI not specified")
	}

	if strings.HasPrefix(settings.PipelineURI, resource.UriScheme) {

		res := f.resManager.GetResource(settings.PipelineURI)

		if res != nil {
			def, ok := res.Object().(*pipeline.Definition)
			if !ok {
				return nil, errors.New("unable to resolve pipeline: " + settings.PipelineURI)
			}
			streamAction.definition = def
		} else {
			return nil, errors.New("unable to resolve pipeline: " + settings.PipelineURI)
		}
	} else {
		def, err := manager.GetPipeline(settings.PipelineURI)
		if err != nil {
			return nil, err
		} else {
			if def == nil {
				return nil, errors.New("unable to resolve pipeline: " + settings.PipelineURI)
			}
		}
		streamAction.definition = def
	}

	streamAction.ioMetadata = streamAction.definition.Metadata()

	if settings.OutputChannel != "" {
		ch := channels.Get(settings.OutputChannel)

		if ch == nil {
			return nil, fmt.Errorf("engine channel `%s` not registered", settings.OutputChannel)
		}

		streamAction.outChannel = ch
	}

	instId := ""

	instLogger := logger

	if log.CtxLoggingEnabled() {
		instLogger = log.ChildLoggerWithFields(logger, log.FieldString("pipelineName", streamAction.definition.Name()), log.FieldString("pipelineId", instId))
	}

	//note: single pipeline instance for the moment
	inst := pipeline.NewInstance(streamAction.definition, instId, settings.GroupBy == "", streamAction.outChannel, instLogger)
	streamAction.inst = inst

	return streamAction, nil
}

type StreamAction struct {
	ioMetadata *metadata.IOMetadata
	definition *pipeline.Definition
	outChannel channels.Channel

	inst    *pipeline.Instance
	groupBy string
}

func (s *StreamAction) Info() *action.Info {
	panic("implement me")
}

func (s *StreamAction) Metadata() *action.Metadata {
	return actionMd
}

func (s *StreamAction) IOMetadata() *metadata.IOMetadata {
	return s.ioMetadata
}

func (s *StreamAction) Run(context context.Context, inputs map[string]interface{}, handler action.ResultHandler) error {

	discriminator := ""

	if s.groupBy != "" {
		//navigate input
		//note: for now groupings are determined by inputs to the action
		value, ok := inputs[s.groupBy]
		if ok {
			discriminator, _ = coerce.ToString(value)
		}
	}

	logger.Debugf("Running pipeline")

	go func() {

		defer handler.Done()
		retData, status, err := s.inst.Run(discriminator, inputs)

		if err != nil {
			handler.HandleResult(nil, err)
		} else {
			handler.HandleResult(retData, err)
		}

		if s.outChannel != nil && status == pipeline.ExecStatusCompleted {
			s.outChannel.Publish(retData)
		}
	}()

	return nil
}
