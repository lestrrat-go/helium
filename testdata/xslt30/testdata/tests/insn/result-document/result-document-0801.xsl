<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test AVT in @output-version of xsl:result-document.-->

   <t:template match="/doc">
	     <t:result-document output-version="{foo}">
		       <out>hello</out>
	     </t:result-document>
   </t:template>
</t:transform>
