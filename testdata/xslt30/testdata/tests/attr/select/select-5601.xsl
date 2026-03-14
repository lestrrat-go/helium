<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#document?>
    <!-- Purpose: Test that document('') refers to this stylesheet, and exploit
     that fact to choose a template dynamically. Idea from Mike Kay. -->

<xsl:template match="/document-element">
  <xsl:variable name="whichtmplt" select="'this'"/>
  <out>
    <xsl:apply-templates select="document('')/*/xsl:template[@name=$whichtmplt]"/>
    <xsl:apply-templates/>
  </out>
</xsl:template>

<xsl:template name="this" match="xsl:template[@name='this']">We are inside.
  <xsl:value-of select="name(.)"/>
</xsl:template>

<xsl:template name="that" match="xsl:template[@name='that']">We are offside.
  <xsl:value-of select="name(.)"/>
</xsl:template>

<xsl:template name="the_other" match="*">We are generic.
  <xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
