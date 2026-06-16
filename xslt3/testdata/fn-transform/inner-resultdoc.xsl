<?xml version="1.0"?>
<xsl:stylesheet version="3.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">
  <xsl:template match="item">
    <principal><xsl:value-of select="."/></principal>
    <xsl:result-document href="secondary.xml">
      <secondary><xsl:value-of select="."/></secondary>
    </xsl:result-document>
  </xsl:template>
</xsl:stylesheet>
