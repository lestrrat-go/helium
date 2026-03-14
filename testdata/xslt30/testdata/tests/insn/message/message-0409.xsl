<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="3.0">
   <!-- Purpose: xsl:message outputs a function item -->
   <t:template match="/">
      <out>
         <t:message select="function($x){$x*2}"/>
      </out>
   </t:template>
</t:transform>
