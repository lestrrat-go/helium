<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Test combination of key() and document() on multiple files. -->

<xsl:key name="bib" match="entry" use="@name" />

<xsl:template match="doc">
 <root>
  <xsl:apply-templates/>
 </root>
</xsl:template>

<xsl:template match="bibref">
  Got a bibref for <xsl:value-of select="."/>...
  <xsl:variable name="lookup" select="."/>
  <xsl:for-each select="document(refdoc)">
    <xsl:apply-templates select="key('bib',$lookup)"/>
  </xsl:for-each>
</xsl:template>

<xsl:template match="refdoc"><!-- suppress default action -->
  <xsl:apply-templates/>
</xsl:template>

</xsl:stylesheet>