<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             default-collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
<!-- Purpose: Test to verify that xsl:for-each-group uses the default collation explicitly specified on the stylesheet element-->

   <t:template match="/"> 
      <out>
Default collation:<t:text>
</t:text>
         <t:for-each-group select="doc/item" group-by="@target">
            <t:value-of select="@num"/>
         </t:for-each-group>
         <t:text>
</t:text>
      </out>
   </t:template>
</t:transform>
