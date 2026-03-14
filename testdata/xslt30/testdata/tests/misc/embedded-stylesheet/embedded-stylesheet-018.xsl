<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that an static error is raised if the xsl:stylesheet 
  				element	contains any non-whitespace text node children. -->

   <t:template match="/">
	     <t:text>Invalid</t:text>
   </t:template>

Invalid text node

</t:transform>
