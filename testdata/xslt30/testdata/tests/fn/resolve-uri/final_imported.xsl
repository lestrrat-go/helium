<?xml version='1.0'?>
<?spec fo#func-resolve-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:template match="four-tag">
  From stylesheet final_imported.xsl: <xsl:value-of select="self::node()"/>
</xsl:template>

</xsl:stylesheet>