<?xml version="1.0"?>
<?spec xslt#number?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<!-- test xsl:number with different values of lang attribute in a single run -->

<xsl:output method="xml" indent="no" encoding="UTF-8"/>

<xsl:template match="doc">
  <out>
    <xsl:apply-templates select=".//note"/>
  </out>
</xsl:template>

<xsl:template match="note">
  <xsl:variable name="temp">
    <xsl:number value="position()" format="Ww" lang="{if (position() mod 2 = 0) then 'en' else 'de'}" />
  </xsl:variable>
  <!-- allow both "twenty-one" and "twenty one" -->
  <n><xsl:value-of select="translate($temp, '-', ' ')"/></n>
</xsl:template>

</xsl:stylesheet>