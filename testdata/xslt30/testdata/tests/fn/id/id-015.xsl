<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec fo#func-id?>
  <!-- Purpose: Test for id(). -->

<xsl:template match="t04">
 <out>
    <xsl:value-of select="id('c')/@id"/>
 </out>
</xsl:template>

</xsl:stylesheet>
