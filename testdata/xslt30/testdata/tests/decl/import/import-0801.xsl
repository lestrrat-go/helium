<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#apply-imports?>
  <!-- Purpose: General test of xsl:apply-imports from spec. -->

<xsl:import href="l.xsl"/>

<xsl:template match="doc">
 <out>
  <xsl:apply-templates/>
 </out>
</xsl:template>

<xsl:template match="tag">
  <div style="border: solid red">
    <xsl:apply-imports/>
  </div>
</xsl:template>

</xsl:stylesheet>