<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:test="http://www.examples.org/test"
             xmlns:t="http://www.w3.org/1999/XSL/Transform"
             exclude-result-prefixes="test"
             version="2.0">
<!-- Purpose: Uses function-available to test the existance of an extension function inside a use-when instruction. -->

   <t:template match="doc">
		    <out>
         <t:element name="a" use-when="function-available('test:evaluate')">
				        <t:value-of select="'Available'"/>
			      </t:element>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
