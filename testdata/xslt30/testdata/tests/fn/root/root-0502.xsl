<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Apply root(node) to a document opened via document() -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
	     <out>
         <t:choose>
	           <t:when test="generate-id(root(document('extradoc.xml'))) =                         generate-id(root(document('extradoc.xml')//c))">
	              <t:text>Success</t:text>
	           </t:when>
	           <t:otherwise>
	              <t:text>FAIL! tt: </t:text>
	              <t:value-of select="generate-id(root(document('extradoc.xml')))"/>
	              <t:text>, tt//c: </t:text>
	              <t:value-of select="generate-id(root(document('extradoc.xml')//c))"/>
	           </t:otherwise>
	        </t:choose>
      </out>
	  </t:template>
</t:transform>
