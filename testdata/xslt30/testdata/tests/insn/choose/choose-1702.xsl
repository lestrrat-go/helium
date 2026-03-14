<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses if expression in the test attribute of an xsl:when instructions. -->

   <t:template match="/">
	     <out>
         <t:choose>
	  		       <t:when test="if (1 &lt; 0) then 1 else if (contains(doc,'bar')) then 2 else 3">
	      		      <t:value-of select="if (1 &lt; 0) then 1 else if (contains(doc,'bar')) then 2 else 3"/>
	  		       </t:when>
	  	     </t:choose>
      </out>
	  </t:template>
</t:transform>
