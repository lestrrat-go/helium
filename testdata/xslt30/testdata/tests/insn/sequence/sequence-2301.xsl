<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">

   <!-- Purpose: xsl:sequence node identity should be preserved -->

   <t:output method="xml" encoding="UTF-8"/>

   <t:template match="/">
      <t:variable name="nodeseq" as="item()*">
         <t:sequence select="(doc/one, doc/two)"/>
      </t:variable>
      <out>
         <t:value-of select="$nodeseq[1] is doc/one"/>
         <t:value-of select="$nodeseq[2] is doc/two"/>
      </out>
   </t:template>
</t:transform>
