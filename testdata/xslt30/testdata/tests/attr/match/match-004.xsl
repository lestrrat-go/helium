<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#patterns?>
  <!-- Purpose: Test of @attrib=value in predicate in match pattern. -->

<xsl:template match="doc">
  <out>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="*[@val=4]">
  <xsl:value-of select="name(.)"/>
</xsl:template>

<xsl:template match="text()"/>

</xsl:stylesheet>
