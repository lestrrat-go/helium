<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test with xsl:result-document and  @doctype-public present 
  				but no @doctype-system present. Show that @doctype-public is ignored. -->

   <t:template match="/">
  	   <t:result-document method="xml" doctype-public="-//BOAG//DTD Websites V1.3//EN">
      	  <out>hello</out>
      </t:result-document>
   </t:template>
</t:transform>
