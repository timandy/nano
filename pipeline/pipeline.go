package pipeline

type pipeline struct {
	inbound  *pipelineChain
	outbound *pipelineChain
}

func (p *pipeline) Inbound() PipelineChain {
	return p.inbound
}

func (p *pipeline) Outbound() PipelineChain {
	return p.outbound
}

func New() Pipeline {
	return &pipeline{
		outbound: &pipelineChain{},
		inbound:  &pipelineChain{}}
}
