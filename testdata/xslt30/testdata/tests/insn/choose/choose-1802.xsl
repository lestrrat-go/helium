<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case where the xsl:when instruction is missing the required test attribute. -->

   <t:template match="/">
		    <out>
         <t:choose>
			         <t:when>
	   			        <t:text>number </t:text>
	  		       </t:when>
	  		       <t:otherwise>
	   			        <t:text>string</t:text>
	  		       </t:otherwise>
	  		    </t:choose>
      </out>
	  </t:template>
</t:transform>
