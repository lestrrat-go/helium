package sax

import "errors"

func New() *SAX2 {
	return &SAX2{}
}

func (s *SAX2) SetDocumentLocator(ctx interface{}, loc DocumentLocator) error {
	if h := s.SetDocumentLocatorHandler; h != nil {
		return h(ctx, loc)
	}
	return nil
}

func (s *SAX2) StartDocument(ctx interface{}) error {
	if h := s.StartDocumentHandler; h != nil {
		return h(ctx)
	}
	return nil
}

func (s *SAX2) EndDocument(ctx interface{}) error {
	if h := s.EndDocumentHandler; h != nil {
		return h(ctx)
	}
	return nil
}

func (s *SAX2) StartElement(ctx interface{}, elem ParsedElement) error {
	if h := s.StartElementHandler; h != nil {
		return h(ctx, elem)
	}
	return nil
}

func (s *SAX2) EndElement(ctx interface{}, elem ParsedElement) error {
	if h := s.EndElementHandler; h != nil {
		return h(ctx, elem)
	}
	return nil
}

func (s *SAX2) CDATABlock(ctx interface{}, data []byte) error {
	if h := s.CDATABlockHandler; h != nil {
		return h(ctx, data)
	}
	return nil
}

func (s *SAX2) Characters(ctx interface{}, data []byte) error {
	if h := s.CharactersHandler; h != nil {
		return h(ctx, data)
	}
	return nil
}

func (s *SAX2) Comment(ctx interface{}, data []byte) error {
	if h := s.CommentHandler; h != nil {
		return h(ctx, data)
	}
	return nil
}

func (s *SAX2) ProcessingInstruction(ctx interface{}, target, data string) error {
	if h := s.ProcessingInstructionHandler; h != nil {
		return h(ctx, target, data)
	}
	return nil
}

func (s *SAX2) InternalSubset(ctx interface{}, name, eid, uri string) error {
	if h := s.InternalSubsetHandler; h != nil {
		return h(ctx, name, eid, uri)
	}
	return nil
}

func (s *SAX2) GetParameterEntity(ctx interface{}, name string) (Entity, error) {
	if h := s.GetParameterEntityHandler; h != nil {
		return h(ctx, name)
	}
	return nil, errors.New("not found")
}
