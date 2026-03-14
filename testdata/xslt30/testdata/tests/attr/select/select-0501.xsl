<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xpath#axes?>
    <!-- Purpose: Test for selecting parent nodes. -->

<xsl:template match="/">
  <out>At the root, should say the name is ():<xsl:apply-templates/></out>
</xsl:template>

<xsl:template match="way-inside">We are way inside.
  <xsl:apply-templates select=".." mode="parent"/>
</xsl:template>

<xsl:template match="* | /" mode="parent">
  name of parent is (<xsl:value-of select="name(.)"/>)
  <xsl:apply-templates select=".." mode="parent"/>
</xsl:template>

</xsl:stylesheet>
