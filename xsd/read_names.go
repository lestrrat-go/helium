package xsd

import (
	"context"
	"errors"

	helium "github.com/lestrrat-go/helium"
	"github.com/lestrrat-go/helium/internal/xmlchar"
)

func (c *compiler) readRequiredTopLevelNCName(ctx context.Context, elem *helium.Element, missingErr, component string, componentError bool) (string, bool, error) {
	name := collapsedAttr(elem, attrName)
	if name == "" {
		if hasAttr(elem, attrName) {
			c.reportInvalidTopLevelNCName(ctx, elem, name, component, componentError)
			return "", false, nil
		}
		return "", false, errors.New(missingErr)
	}
	if !xmlchar.IsValidNCName(name) {
		c.reportInvalidTopLevelNCName(ctx, elem, name, component, componentError)
		return "", false, nil
	}
	return name, true, nil
}

func (c *compiler) reportInvalidTopLevelNCName(ctx context.Context, elem *helium.Element, name, component string, componentError bool) {
	if c.filename == "" {
		return
	}
	msg := "The value '" + name + "' of attribute 'name' is not a valid 'xs:NCName'."
	if componentError {
		c.schemaError(ctx, schemaComponentError(c.diagSource(), elem.Line(), elem.LocalName(), component, msg))
		return
	}
	c.schemaError(ctx, schemaParserError(c.diagSource(), elem.Line(), elem.LocalName(), component, msg))
}
