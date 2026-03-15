<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

<?spec xslt#document?>
    <!-- Purpose: Test that document('') refers to this stylesheet, and exploit
     that fact to have data file select which template to use. -->
     
  <!-- Priorities added by MHK to suppress warnings -->

<xsl:template match="/document-element">
  <xsl:variable name="whichtmplt" select="chooser"/>
  <out>
    <xsl:apply-templates select="document('')/*/xsl:template[@name=$whichtmplt]"/>
  </out>
</xsl:template>

<xsl:template name="this" match="xsl:template" priority="2">This template.
  <xsl:apply-templates select="document('select-68a.xml')//inside" mode="thismode"/>
</xsl:template>

<xsl:template name="that" match="xsl:template" priority="3">That template.
  <xsl:apply-templates select="document('select-68a.xml')//inside" mode="thatmode"/>
</xsl:template>

<xsl:template match="inside" mode="thismode">We are inside.
  <xsl:value-of select="@xx"/>
</xsl:template>

<xsl:template match="inside" mode="thatmode">We got inside.
  <xsl:value-of select="@flag"/>
</xsl:template>

<xsl:template match="inside">We got inside, but modeless.
  <xsl:value-of select="."/>
</xsl:template>

<xsl:template name="the_other" match="*">We are generic.
  <xsl:value-of select="name(.)"/>
</xsl:template>

</xsl:stylesheet>
