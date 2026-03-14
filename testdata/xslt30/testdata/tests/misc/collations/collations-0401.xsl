<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform"
             version="2.0"
             default-collation="http://www.w3.org/2005/xpath-functions/collation/codepoint">
<!-- Purpose: Test to verify that xsl:key uses the default collation explicitly specified in the stylesheet element when no collation attribute is used-->

   <t:key name="key1" match="item" use="@target"/>

   <t:template match="/"> 
      <out>
         <t:value-of select="key('key1', 'a')/@num"/>
      </out>
   </t:template>
</t:transform>
