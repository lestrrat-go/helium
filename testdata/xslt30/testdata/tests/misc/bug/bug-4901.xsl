<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0">

<?spec fo#func-name?>
<!-- BUG (reported but not reproduced) name() doesn't work on attribute nodes. -->

<xsl:template match="/">
  <testing>
    <m><xsl:value-of select="doc/@mary"/></m>
    <xsl:for-each select="//@*">
      <a><xsl:value-of select="name()"/></a>
    </xsl:for-each>
  </testing>
</xsl:template>

</xsl:stylesheet>
