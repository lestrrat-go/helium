<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Give the select attribute of xsl:value-of a path expression that selects more than one node. No @separator is given. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/">
      <out>
         <t:value-of select="//item" version="1.0"/>
      </out>
   </t:template>
</t:transform>
