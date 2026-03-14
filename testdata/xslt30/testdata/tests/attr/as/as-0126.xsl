<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test type of global xsl:variable without @select and @as="element()". 
  				Sequence constructor contains an xsl:element instruction or an LRE. -->

   <t:variable name="var1" as="element()">
		    <t:element name="el1">element value</t:element>
   </t:variable>

   <t:variable name="var2" as="element()">
		    <item>hello</item>
   </t:variable>

   <t:template match="/doc">
      <out>
         <t:value-of select="$var1 instance of element()"/>
         <t:value-of select="$var2 instance of element()"/>
      </out>
   </t:template>
</t:transform>
