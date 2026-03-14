<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test type of local xsl:param that has @as="element()" with: 
				- @select selecting an element node from input file
				- no @select and sequence constructor has xsl:element
				- no @select and sequence constructor has an LRE -->

   <t:template match="/doc">

      <t:param name="par1" select="item" as="element()"/>

      <t:param name="par2" as="element()">
		       <t:element name="el1">element value</t:element>
      </t:param>

      <t:param name="par3" as="element()">
		       <item>hello</item>
      </t:param>
      <out>
         <t:value-of select="$par1 instance of element()"/>
         <t:value-of select="$par2 instance of element()"/>
         <t:value-of select="$par3 instance of element()"/>
      </out>
   </t:template>
</t:transform>
