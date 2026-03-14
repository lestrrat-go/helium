<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: Test with xsl:result-document where value of @standalone=" false ".-->

   <t:template match="/">
      <t:result-document method="xml" standalone=" false ">
      	  <out>hello</out>
      </t:result-document>
   </t:template>
</t:transform>
