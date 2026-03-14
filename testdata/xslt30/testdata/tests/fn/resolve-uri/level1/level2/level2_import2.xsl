<?xml version='1.0'?>
<?spec fo#func-resolve-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:import href="level3/level3_import1.xsl"/>

<xsl:template match="two-tag">
  From stylesheet level2_import2.xsl: <xsl:value-of select="self::node()"/>
  importing level3_import1.xsl
</xsl:template>

</xsl:stylesheet>