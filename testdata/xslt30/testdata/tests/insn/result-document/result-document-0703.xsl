<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test AVT in @standalone of xsl:result-document.-->

   <t:template match="/">
  	   <t:result-document method="xml" standalone="{doc/foo}">
      	  <out>hello</out>
      </t:result-document>
   </t:template>
</t:transform>
