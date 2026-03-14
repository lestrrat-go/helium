<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses if expression in the test attribute of an xsl:if instructions. -->

   <t:template match="/">
	     <out>
         <t:if test="if (contains(doc,'oo')) then 1 else 0">
	  		       <t:value-of select="if (contains(doc,'oo')) then 1 else 0"/>
	 	      </t:if>
      </out>
	  </t:template>
</t:transform>
