<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Attribute encountered before first copied node must not be copied. -->
  <!-- Discretionary: add-attribute-to-non-element="ignore" -->

<xsl:output method="xml" encoding="UTF-8"/>

<xsl:template match="test">
  <xsl:document>
    <xsl:copy-of select="*|@*"/>
  </xsl:document>
</xsl:template>

</xsl:stylesheet>