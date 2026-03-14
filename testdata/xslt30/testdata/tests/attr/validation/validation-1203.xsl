<?xml version="1.0"?>
<xsl:stylesheet version="3.0"
xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
xmlns:t="http://ns.example.com/val12/"
>

<!-- PURPOSE: effect of implicit validation="preserve" on the nilled property -->
<?spec fo#func-nilled?>

<xsl:template match="/">
<z>
  <xsl:variable name="c" as="document-node()" select="copy-of(.)"/>
  <xsl:variable name="s" as="document-node()" select="snapshot(.)"/>
  <xsl:apply-templates select="($c, $s)//t:test"/>
</z>
</xsl:template>

<xsl:template match="t:test">
  <a><xsl:value-of select="nilled(.)" /></a>
</xsl:template>

</xsl:stylesheet>
