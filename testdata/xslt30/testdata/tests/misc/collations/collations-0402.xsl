<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Test to verify that xsl:key uses the explicit collation specified in its collation attribute-->

   <t:key name="key1"
          match="item"
          use="@target"
          collation="http://www.w3.org/2005/xpath-functions/collation/codepoint"/>

   <t:template match="/"> 
      <out>
         <t:value-of select="key('key1', 'a')/@num"/>
      </out>
   </t:template>
</t:transform>
