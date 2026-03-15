<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test case to verify that if the select attribute is present for an xsl:comment instruction
  				the strings specified as its value are concatenated with a single space between adjacent 
  				strings.-->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
  		  <out>
         <t:comment select="doc/book"/>
         <t:comment select="('a1','','a2','a3')"/>
      </out>
	  </t:template>

   <t:template match="node()">
  		  <t:apply-templates/>
	  </t:template>
</t:transform>
