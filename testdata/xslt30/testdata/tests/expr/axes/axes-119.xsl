<?xml version="1.0"?>
<xsl:stylesheet version="2.0" xmlns:xsl="http://www.w3.org/1999/XSL/Transform">

<?spec xpath#abbrev?>
    <!-- Purpose: Test of //@ sequences -->

<xsl:output method="xml" encoding="utf-8"/>

<xsl:template match="/">
  <out><xsl:text>&#10;</xsl:text>
    <all-attribs><xsl:value-of select="count(//@*)"/></all-attribs><xsl:text>&#10;</xsl:text>
    <all-titles><xsl:value-of select="count(//@title)"/></all-titles><xsl:text>&#10;</xsl:text>
    <all-sect-attribs><xsl:value-of select="count(//section//@*)"/></all-sect-attribs><xsl:text>&#10;</xsl:text>
    <all-sect-titles><xsl:value-of select="count(//section//@title)"/></all-sect-titles><xsl:text>&#10;</xsl:text>
    <!-- The above two, respectively, must equal the sums of sect-*-attribs and sect-*-titles below. -->
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template match="chapter">
  <chap-attribs><xsl:value-of select="count(.//@*)"/></chap-attribs><xsl:text>&#10;</xsl:text>
  <chap-titles><xsl:value-of select="count(.//@title)"/></chap-titles><xsl:text>&#10;</xsl:text>
  <!-- Rather than iterate, we want to have a sub-element name lead the path expression. -->
  <sect-1-attribs><xsl:value-of select="count(section[1]//@*)"/></sect-1-attribs><xsl:text>&#10;</xsl:text>
  <sect-1-titles><xsl:value-of select="count(section[1]//@title)"/></sect-1-titles><xsl:text>&#10;</xsl:text>
  <sect-2-attribs><xsl:value-of select="count(section[2]//@*)"/></sect-2-attribs><xsl:text>&#10;</xsl:text>
  <sect-2-titles><xsl:value-of select="count(section[2]//@title)"/></sect-2-titles><xsl:text>&#10;</xsl:text>
  <sect-3-attribs><xsl:value-of select="count(section[3]//@*)"/></sect-3-attribs><xsl:text>&#10;</xsl:text>
  <sect-3-titles><xsl:value-of select="count(section[3]//@title)"/></sect-3-titles><xsl:text>&#10;</xsl:text>
</xsl:template>

</xsl:stylesheet>