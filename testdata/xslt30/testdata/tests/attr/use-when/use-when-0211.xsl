<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses use-when attribute with a value of false on an xsl:template.  -->

   <t:template match="element()">
  	   <out>
         <t:copy>
   		       <t:apply-templates/>
    	    </t:copy>
      </out>
  </t:template>

   <t:template match="a | b" use-when="false()">
	<print>
         <t:next-match/>
    </print>
   </t:template>
</t:transform>
