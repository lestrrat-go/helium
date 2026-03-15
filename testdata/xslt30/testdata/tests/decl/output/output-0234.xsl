<?xml version="1.0" encoding="UTF-8"?>
<xsl:transform xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
<!-- Purpose: With XML output, DOCTYPE must come "immediately before the first element".-->

   <xsl:output method="xml" indent="no" doctype-public="//PUBLIC//" doctype-system="system.dtd"/>

   <xsl:template match="/">
      <xsl:comment>This should precede the DOCTYPE declaration</xsl:comment>
      <xsl:processing-instruction name="pi">R squared</xsl:processing-instruction>
      <out>
         <in/>
      </out>
   </xsl:template>
</xsl:transform>
