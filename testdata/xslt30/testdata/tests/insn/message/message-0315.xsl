<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Tests using xsl:next-match inside an xsl:message instruction. -->

   <t:template match="docs">
	     <out>
         <t:message>
		    	     <t:next-match/>
		       </t:message>
      </out>
	  </t:template>

   <t:template match="*">Text in match=foo</t:template>
</t:transform>
