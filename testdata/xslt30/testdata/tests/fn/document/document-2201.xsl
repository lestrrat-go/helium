<?xml version="1.0"?>
<xsl:stylesheet xmlns:xsl="http://www.w3.org/1999/XSL/Transform" version="2.0">

  <!-- Purpose: Compare two evaluation sequences for variable containing
    node-set returned from document() call. -->


  <xsl:param name="terms" select="document('doc16.xml')"/>

  <xsl:template match="/">
    <out>
      <xsl:call-template name="gentext">
        <xsl:with-param name="key" select="'TableofContents'"/>
        <xsl:with-param name="lang" select="'en'"/>
      </xsl:call-template>
    </out>
  </xsl:template>

  <xsl:template name="gentext">
    <xsl:param name="key" select="local-name(.)"/>
    <xsl:param name="lang" select="FOO"/>
    <xsl:variable name="lookup"
      select="($terms/table/expand[@language=$lang]/gentext[@key=$key])[1]"/>
    <xsl:variable name="l10n.name" select="$lookup/@text"/>
    <xsl:element name="key">
      <xsl:value-of select="$key"/>
    </xsl:element>
    <xsl:element name="lang">
      <xsl:value-of select="$lang"/>
    </xsl:element>
    <xsl:element name="var">
      <xsl:value-of select="$l10n.name"/>
    </xsl:element>
    <xsl:element name="text">
      <xsl:value-of select="$lookup/@text"/>
    </xsl:element>
  </xsl:template>

</xsl:stylesheet>
