<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:test="http://www.examples.org/test"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="test"
             version="2.0">
<!-- Purpose: Test to verify that an error is raised if the name of the extension function does not evaluate to a string that is a valid QName.-->

   <t:template match="doc">
		    <out>
         <t:value-of select="'error'" use-when="function-available('test:foo?')"/>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
