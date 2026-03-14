<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <!-- Purpose: d-o-e is ignored when writing a text node with a nested xsl:value-of instruction -->
  
  <xsl:output omit-xml-declaration="yes"/>

<xsl:template match="/">
  <out>
    <xsl:value-of>
      <xsl:apply-templates select="'&lt;'" mode="doe"/>
    </xsl:value-of>
    <xsl:apply-templates select="'&amp;lt;'" mode="doe"/>
  </out>
</xsl:template>
  
  <xsl:template match="." mode="doe" as="text()">
    <xsl:value-of select="." disable-output-escaping="true"/>
  </xsl:template>

</xsl:stylesheet>
