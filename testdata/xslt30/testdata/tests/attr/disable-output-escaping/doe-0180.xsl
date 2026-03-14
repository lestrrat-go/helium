<?xml version="1.0"?> 
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="3.0">
  <xsl:output method="xml" encoding="UTF-8" indent="no"/>

  <!-- Purpose: d-o-e is ignored when writing an attribute node -->
  
  <xsl:output omit-xml-declaration="yes"/>

<xsl:template match="/">
  <out>
    <xsl:attribute name="special">
      <xsl:apply-templates select="'&lt;'" mode="doe"/>
    </xsl:attribute>
    <xsl:apply-templates select="'&amp;lt;'" mode="doe"/>
  </out>
</xsl:template>
  
  <xsl:template match="." mode="doe" as="text()">
    <xsl:value-of select="." disable-output-escaping="true"/>
  </xsl:template>

</xsl:stylesheet>
