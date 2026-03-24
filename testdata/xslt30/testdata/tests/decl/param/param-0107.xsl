<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
   <!-- Purpose: Tests the shalowing of local variables.  -->

   <t:param name="y" select="'g'"/>

   <t:template match="/">
      <out>
         <t:call-template name="x"/>
         <t:call-template name="y"/>
      </out>
   </t:template>

   <t:template name="x">
      <t:param name="y" select="'a'"/>
      <p1>
         <a1>
            <t:value-of select="$y"/>
         </a1>
         <t:variable name="y" select="'b'"/>
         <b1>
            <t:value-of select="$y"/>
         </b1>
         <t:variable name="y" select="concat($y,'c')"/>
         <c1>
            <t:value-of select="$y"/>
         </c1>
         <t:variable name="y" select="($y,(for $y in ('x','y','z') return $y))"/>
         <d1>
            <t:value-of select="$y" separator=","/>
         </d1>
      </p1>
      <e1>
         <t:value-of select="$y"/>
      </e1>
   </t:template>

   <t:template name="y">
      <p2>
         <a2>
            <t:value-of select="$y"/>
         </a2>
         <t:variable name="y" select="'b'"/>
         <b2>
            <t:value-of select="$y"/>
         </b2>
         <t:variable name="y" select="concat($y,'c')"/>
         <c2>
            <t:value-of select="$y"/>
         </c2>
         <t:variable name="y" select="($y,(for $y in ('x','y','z') return $y))"/>
         <d2>
            <t:value-of select="$y" separator=","/>
         </d2>
      </p2>
      <e2>
         <t:value-of select="$y"/>
      </e2>
   </t:template>
</t:transform>
