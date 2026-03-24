<?xml version="1.0" encoding="ISO-8859-1"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-namespace-uri?>
<?spec fo#func-local-name?>
<?spec fo#func-name?>

    <!-- Purpose: Test name functions on non-namespaced attribute node. -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:text>name|</xsl:text><xsl:value-of select="name(a/@at1)"/><xsl:text>|</xsl:text>
    <xsl:text>namespace-uri|</xsl:text><xsl:value-of select="namespace-uri(a/@at1)"/><xsl:text>|</xsl:text>
    <xsl:text>local-name|</xsl:text><xsl:value-of select="local-name(a/@at1)"/><xsl:text>|</xsl:text>
  </out>
</xsl:template>

</xsl:stylesheet>