<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Apply root(node) to a temporary tree -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:variable name="tt">
	     <a xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
         <b>
            <c/>
         </b>
      </a>
	  </t:variable>

   <t:template match="/">
	     <out>
         <t:choose>
	           <t:when test="generate-id(root($tt)) = generate-id(root($tt//c))">
	              <t:text>Success</t:text>
	           </t:when>
	           <t:otherwise>
	              <t:text>FAIL! tt: </t:text>
	              <t:value-of select="generate-id(root($tt))"/>
	              <t:text>, tt//c: </t:text>
	              <t:value-of select="generate-id(root($tt//c))"/>
	           </t:otherwise>
	        </t:choose>
      </out>
	  </t:template>
</t:transform>
