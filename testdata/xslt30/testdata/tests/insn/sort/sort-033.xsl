<?xml version="1.0"?>
<?spec xslt#sorting?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- FileName: SORT26 -->
  <!-- Document: http://www.w3.org/TR/xslt -->
  <!-- DocVersion: 19991116 -->
  <!-- Section: 10 -->
    <!-- Purpose: Test for xsl:sort on apply-templates that does not have a select attribute. -->

<xsl:template match="w3cgroup">
  <out>
    <xsl:apply-templates>
      <xsl:sort select="name/last | primary/name/last" order="ascending"/>
    </xsl:apply-templates>
  </out>
</xsl:template>

<xsl:template match="member">
  <xsl:value-of select="primary/name/first"/><xsl:text> </xsl:text>
  <xsl:value-of select="primary/name/last"/><xsl:text>; </xsl:text>
</xsl:template>

<xsl:template match="chair" priority="2">
  Chair: <xsl:value-of select="name/first"/><xsl:text> </xsl:text>
  <xsl:value-of select="name/last"/><xsl:text>; </xsl:text>
</xsl:template>

<xsl:template match="w3c-contact" priority="2">
  W3C: <xsl:value-of select="name/first"/><xsl:text> </xsl:text>
  <xsl:value-of select="name/last"/><xsl:text>; </xsl:text>
</xsl:template>

</xsl:stylesheet>
