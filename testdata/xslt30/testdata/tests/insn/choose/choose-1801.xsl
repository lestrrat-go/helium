<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case where the xsl:if instruction does not contain the required test attribute. -->

   <t:template match="/">
		    <out>
         <t:if>
	   			     <t:text>number </t:text>
	  		    </t:if>
         <t:if test="'a'='a'">
	   			     <t:text>string</t:text>
	  		    </t:if>
      </out>
	  </t:template>
</t:transform>
