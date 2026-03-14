<?xml version="1.0" encoding="UTF-8"?>
<t:transform xmlns:t="http://www.w3.org/1999/XSL/Transform" version="2.0">
<!-- Purpose: Use version flag on xsl:namespace to singularize AVT; error if BC not supported. -->

   <t:output method="xml" encoding="UTF-8" indent="no"/>

   <t:template match="/" version="1.0">
      <out>
         <t:namespace name="{doc/item}" select="'http://www.example.ca'" version="1.0"/>
      </out>
   </t:template>
</t:transform>
