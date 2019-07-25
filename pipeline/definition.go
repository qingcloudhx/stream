package pipeline

import (
	"github.com/qingcloudhx/core/activity"
	"github.com/qingcloudhx/core/data/mapper"
	"github.com/qingcloudhx/core/data/metadata"
	"github.com/qingcloudhx/core/data/resolve"
	"github.com/qingcloudhx/core/support"
	"github.com/qingcloudhx/core/support/log"
)

type DefinitionConfig struct {
	Name     string               `json:"name"`
	Metadata *metadata.IOMetadata `json:"metadata"`
	Stages   []*StageConfig       `json:"stages"`
}

func NewDefinition(config *DefinitionConfig, mf mapper.Factory, resolver resolve.CompositeResolver) (*Definition, error) {

	def := &Definition{name: config.Name, metadata: config.Metadata}

	for _, sconfig := range config.Stages {
		stage, err := NewStage(sconfig, mf, resolver)

		if err != nil {
			return nil, err
		}

		def.stages = append(def.stages, stage)
	}

	return def, nil
}

type Definition struct {
	name     string
	stages   []*Stage
	metadata *metadata.IOMetadata
}

// Metadata returns IO metadata for the pipeline
func (d *Definition) Metadata() *metadata.IOMetadata {
	return d.metadata
}

func (d *Definition) Name() string {
	return d.name
}

func (d *Definition) Cleanup() error {
	for _, stage := range d.stages {
		if !activity.IsSingleton(stage.act) {
			if needsCleanup, ok := stage.act.(support.NeedsCleanup); ok {
				err := needsCleanup.Cleanup()
				if err != nil {
					log.RootLogger().Warnf("Error cleaning up activity '%s' in pipeline '%s' : ", activity.GetRef(stage.act), d.name, err)
				}
			}
		}
	}

	return nil
}
