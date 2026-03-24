<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests to verify that an error is raised if if there is no namespace declaration 
  				in scope for the prefix of the QName used as the name of the extension function.-->

   <t:template match="doc">
		    <out>
         <t:value-of select="'error'" use-when="function-available('test:foo')"/>
      </out>
	  </t:template>

   <t:template match="text()"/>
</t:transform>
