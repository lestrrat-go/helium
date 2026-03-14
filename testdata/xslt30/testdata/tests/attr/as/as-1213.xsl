<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test type of local xsl:param that has @as="attribute()" with: 
				- @select selecting an attribute node from input file
				- no @select and sequence constructor has xsl:attribute-->

   <t:template match="/doc">

      <t:param name="par1" select="item/@attrib" as="attribute()"/>

      <t:param name="par2" as="attribute()">
		       <t:attribute name="el1">element value</t:attribute>
      </t:param>

      <out>
         <t:value-of select="$par1 instance of attribute()"/>
         <t:value-of select="$par2 instance of attribute()"/>
      </out>
   </t:template>
</t:transform>
