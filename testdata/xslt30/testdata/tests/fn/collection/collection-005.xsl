<?xml version="1.0" encoding="UTF-8"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">


  <!-- test that xsl:strip-space applies to the results of the collection() function -->

  <xsl:mode name="m2" on-no-match="shallow-skip"/>

  <xsl:strip-space elements="a"/>
  <xsl:preserve-space elements="b"/>

  <xsl:template match="/">
    <out>
      <xsl:apply-templates select="collection('collection-004.xml')" mode="m2"/>
    </out>
  </xsl:template>

  <xsl:template match="a[not(@id)] | b" mode="m2">
    <xsl:copy-of select="."/>
  </xsl:template>

</xsl:stylesheet>
