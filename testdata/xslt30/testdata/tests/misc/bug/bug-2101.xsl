<?xml version="1.0" encoding="UTF-8"?>

<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform"
                version="2.0" >

<?spec xpath#node-tests?>
<!-- BUG:  5.3.2/020 -->
<!-- DESCRIPTION:
    When Saxon is used with Xerces (or any other SAX2 parser that does not internalize strings
    by default), attribute names used in a match pattern may go unrecognized. -->

<xsl:template match="/">
  <xsl:apply-templates />
</xsl:template>

<xsl:template match="@att_1">
  <xsl:attribute name="att_2">
    <xsl:value-of select="." />
  </xsl:attribute>
</xsl:template>

<xsl:template match="*">
  <bob>
    <xsl:apply-templates select="@*|*|text()" />
  </bob>
</xsl:template>

</xsl:stylesheet>

