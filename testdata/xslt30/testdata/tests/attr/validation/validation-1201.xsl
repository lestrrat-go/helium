<?xml version="1.0"?>
<xsl:stylesheet version="2.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val12/"
>

<!-- PURPOSE: show the nilled() function -->
<?spec fo#func-nilled?>

<xsl:template match="/">
<z>
  <xsl:apply-templates/>
  <xsl:apply-templates select="doc('validation-20.xml')/*"/>
</z>
</xsl:template>

<xsl:template match="t:test">
  <a><xsl:value-of select="nilled(.)" /></a>
</xsl:template>

</xsl:stylesheet>
