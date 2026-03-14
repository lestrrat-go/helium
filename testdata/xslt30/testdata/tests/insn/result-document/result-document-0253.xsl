<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test with xsl:result-document and @omit-xml-declaration=" 1 ", 
  				@method="xml". Declaration must be omitted from the result tree.-->

   <t:template match="/">
  	   <t:result-document method="xml" omit-xml-declaration=" 1 ">
      	  <out>hello</out>
      </t:result-document>
   </t:template>
</t:transform>
