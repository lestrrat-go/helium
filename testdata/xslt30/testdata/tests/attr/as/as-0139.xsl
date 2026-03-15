<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Test type of global xsl:variable that has @as="item()?" and no @select, 
  				the sequence constructor contains one of: - nothing, LRE, xsl:value-of,
  				xsl:element, xsl:attribute-->

   <t:variable name="var1" as="item()?"> </t:variable>

   <t:variable name="var2" as="item()?">
      <i1>hello</i1>
   </t:variable>

   <t:variable name="var3" as="item()?">
      <t:value-of select="'my'"/>
   </t:variable>

   <t:variable name="var4" as="item()?">
      <t:element name="el1">1</t:element>
   </t:variable>

   <t:variable name="var5" as="item()?">
      <t:attribute name="at1">blue</t:attribute>
   </t:variable>

   <t:template match="/doc">
      <out>
         <e1>
            <t:value-of select="$var1 instance of item()*"/>
         </e1>
         <e2>
            <t:value-of select="$var2 instance of item()*"/>
         </e2>
         <e3>
            <t:value-of select="$var3 instance of item()*"/>
         </e3>
         <e4>
            <t:value-of select="$var4 instance of item()*"/>
         </e4>
         <e5>
            <t:value-of select="$var5 instance of item()*"/>
         </e5>
      </out>
   </t:template>
</t:transform>
