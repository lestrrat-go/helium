<?xml version='1.0'?>
<?spec fo#func-resolve-uri?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<xsl:import href="../../../final_imported.xsl"/>

<xsl:template match="three-tag">
  From  stylesheet level3_import1.xsl: <xsl:value-of select="self::node()"/>
  importing final_imported.xsl
</xsl:template>


</xsl:stylesheet>