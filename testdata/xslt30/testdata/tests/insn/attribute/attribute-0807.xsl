<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case that uses xsl:attribute instruction to create an attribute without having the REQUIRED name.  -->

   <t:template match="doc">
		    <t:text>
</t:text>
		    <out>
         <t:text>
</t:text>
         <t:element name="elem">
			         <t:attribute select="1 to 5"/>
		       </t:element>
         <t:text>
</t:text>
      </out>
	     <t:apply-templates/>
  
   </t:template>
</t:transform>
