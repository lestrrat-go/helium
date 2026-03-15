<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with @output-version="1.0" on xsl:result-document.-->

   <t:template match="/">
	     <t:result-document output-version="1.0">
		       <out>hello</out>
	     </t:result-document>
   </t:template>
</t:transform>
