<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Test combination of key() and document() as suggested in spec. -->

<xsl:key name="bib" match="entry" use="@name" />

<xsl:template match="doc">
 <root>
  <xsl:apply-templates/>
 </root>
</xsl:template>

<xsl:template match="bibref">
  <xsl:variable name="lookup" select="."/>
  <xsl:for-each select="document('bib.xml')">
    <xsl:apply-templates select="key('bib',$lookup)"/>
  </xsl:for-each>
</xsl:template>

</xsl:stylesheet>
