<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Testcase to verify that xsl:for-each-group uses the collation explicitly specified in its collation attribute-->

   <t:template match="/"> 
      <out>
Unicode codepoint collation:<t:text>
</t:text>
         <t:for-each-group select="doc/item"
                           group-by="@target"
                           collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
            <t:value-of select="@num"/>
         </t:for-each-group>
         <t:text>
</t:text>
      </out>
   </t:template>
</t:transform>
