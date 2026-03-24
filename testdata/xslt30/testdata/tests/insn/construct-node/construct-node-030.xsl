<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case using xsl:value-of inside and xsl:comment instruction to specify separator
  				for comment text.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
  		  <out>
         <t:comment>
  			       <t:value-of select="doc/book" separator="**"/>
  		     </t:comment>
         <t:comment>
  			       <t:value-of separator="*">
	  			         <t:text>b1</t:text>
	  			         <t:text>b2</t:text>
	  			         <t:text>b3</t:text>
	  			         <t:text>b4</t:text>
  			       </t:value-of>
  		     </t:comment>
      </out>
	  </t:template>

   <t:template match="node()">
  		  <t:apply-templates/>
	  </t:template>
</t:transform>
