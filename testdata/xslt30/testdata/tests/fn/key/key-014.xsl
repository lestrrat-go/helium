<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#keys?>
    <!-- Purpose: Test for xsl:key and key() with a leading underscore in the name. -->

<xsl:key name="_my_key" match="div" use="title"/>

<xsl:template match="doc">
  <out>
    <xsl:value-of select="key('_my_key', 'Introduction')/p"/><xsl:text> </xsl:text>
    <xsl:value-of select="key('_my_key', 'Stylesheet Structure')/p"/><xsl:text> </xsl:text>
    <xsl:value-of select="key('_my_key', 'Expressions')/p"/>
  </out>
</xsl:template>

</xsl:stylesheet>
