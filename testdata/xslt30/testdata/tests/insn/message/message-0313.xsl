<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses unparsed-text() in the select attribute of
  				an xsl:message instruction (The message text is retrieved from a text file).  -->

   <t:template match="docs">
   		<out>
         <t:message select="unparsed-text('sample-plain.txt')"/>
      </out>
	  </t:template>
</t:transform>
