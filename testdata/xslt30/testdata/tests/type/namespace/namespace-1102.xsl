<?xml version="1.0" encoding="iso-8859-1"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<!-- attribute must have prefix -->

<!-- submitted by Eric Bratton, shows bug in Saxon 6.5 -->

  <xsl:template match="doc">
  <newdoc6 xmlns="http://NotDefault/">
    <xsl:attribute name="third" namespace="http://NotDefault/">3rd</xsl:attribute>
  </newdoc6>
  </xsl:template>

</xsl:stylesheet>
