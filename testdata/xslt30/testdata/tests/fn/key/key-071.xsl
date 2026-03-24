<?xml version="1.0" encoding="UTF-8"?>
<?spec xslt#keys?>
<!-- test xsl:key with a multi-valued variable in a key() pattern -->

<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
  xmlns:xs="http://www.w3.org/2001/XMLSchema"
  exclude-result-prefixes="xs">

  <xsl:key name="k" match="item" use="@id"/>
  <xsl:param name="x" select="('a', 'd', 'f')"/>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="//item"/>
    </out>
  </xsl:template>

  <xsl:template match="key('k', $x)">
    <xsl:copy-of select="."/>
  </xsl:template>

  <xsl:template match="*"/>

</xsl:stylesheet>
