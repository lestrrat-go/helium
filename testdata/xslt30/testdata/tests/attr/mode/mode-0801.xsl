<?xml version="1.0"?>
<?spec xslt#modes?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Within a given mode, there are situations when more than one template
    will match. Normal conflict-resolution rules should apply. -->

<xsl:template match="/">
  <out>
    <xsl:apply-templates select="sss/sss" mode="c"/>
    <xsl:apply-templates select="sss/sss" mode="d"/>
    <xsl:apply-templates select="sss//i" mode="c"/>
    <xsl:apply-templates select="sss//i" mode="d"/>
    <xsl:apply-templates select="/sss/sss/i" mode="c"/>
    <xsl:apply-templates select="/sss/sss/i" mode="d"/>
  </out>
</xsl:template>

<xsl:template match="sss//*" mode="d">
 !Any descendant of any sss!
</xsl:template>

<xsl:template match="/sss//*" mode="d">
 +Any descendant of root sss+
</xsl:template>

<xsl:template match="sss/*" mode="c">
 -Any child of any sss-
</xsl:template>

<xsl:template match="/sss/*" mode="c">
 -Any child of root sss-
</xsl:template>

</xsl:stylesheet>
